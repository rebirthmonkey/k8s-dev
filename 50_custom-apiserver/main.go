package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"

	api "50_custom-apiserver/pkg/apis/restaurant"
	apiv1 "50_custom-apiserver/pkg/apis/restaurant/v1alpha1"
	apiv2 "50_custom-apiserver/pkg/apis/restaurant/v1beta1"
	"50_custom-apiserver/pkg/apiserver"
	"50_custom-apiserver/pkg/apiserver/client"
	authn2 "50_custom-apiserver/pkg/apiserver/filters/authn"
	"50_custom-apiserver/pkg/apiserver/informers"
	"50_custom-apiserver/pkg/utils"
	"50_custom-apiserver/pkg/version"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	logs.InitLogs()
	defer logs.FlushLogs()

	var insecureAddr string
	var insecurePort int
	var backend string
	var etcdServers []string
	var fileRootpath string
	var requestTimeout time.Duration
	var tokenAuthFile string
	var unrestrictedUpdate bool
	var kubectlDisabled bool
	var kubectlEphemeralToken bool
	var kubectlEphemeralDuration time.Duration
	var authnAllowLocalhost bool
	var apiextUrl string
	var portable bool

	pflag.StringVar(&insecureAddr, "insecure-addr", "0.0.0.0", "The addr on which to serve unsecured access.")
	pflag.IntVar(&insecurePort, "insecure-port", 6080, "The port on which to serve unsecured access.")
	pflag.StringVar(&backend, "backend", "etcd", "Storage backend, etcd or file.")
	pflag.StringSliceVar(&etcdServers, "etcd-servers", nil,
		"List of etcd servers to connect with (scheme://ip:port), comma separated. use with etcd backend.")
	pflag.StringVar(&fileRootpath, "file-rootpath", "", "Storage root path. use with file backend.")
	pflag.DurationVar(&requestTimeout, "request-timeout", time.Minute,
		"Timeout for requests which match the LongRunningFunc predicate.")
	pflag.StringVar(&tokenAuthFile, "token-auth-file", "",
		"If set, the file that will be used to secure the api server via token authentication.")
	pflag.BoolVar(&unrestrictedUpdate, "unrestricted-update", false,
		"If true, allows subresources to be updates when PUTing the main resource. debug purpose only")
	pflag.BoolVar(&kubectlDisabled, "kubectl-disabled", false,
		"If true, prevents kubectl from accessing the api server")
	pflag.BoolVar(&kubectlEphemeralToken, "kubectl-ephemeral-token", false,
		"If true, kubectl can only authenticate the api server with an ephemeral token")
	pflag.DurationVar(&kubectlEphemeralDuration, "kubectl-ephemeral-duration", time.Hour,
		"Maximum remaining time until ephemeral token expiration for kubectl")
	pflag.BoolVar(&authnAllowLocalhost, "authn-allow-localhost", false,
		"whether to accept request from localhost with no token provided")
	pflag.StringVar(&apiextUrl, "apiext-url", "http://127.0.0.1:6084", "URL of the apiext server")
	pflag.BoolVar(&portable, "portable", false, "If run the api server in portable mode")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	klog.Infof("custom version: %s", version.Info())
	customPrefix := "/custom"

	apiextURL, err := url.Parse(apiextUrl)
	if err != nil {
		klog.Fatal(err)
	}

	var staticToken string
	if tokenAuthFile != "" {
		staticTokenAuth, err := authn2.NewCSV(tokenAuthFile)
		if err != nil {
			klog.Fatal(err)
		}
		staticToken = staticTokenAuth.GetToken(authn2.TeleportUID)
	}

	informer, err := informers.New(utils.LocalhostIP, insecurePort, staticToken, time.Hour)
	if err != nil {
		klog.Fatal(err)
	}

	config, err := apiserver.Config{
		SecurePort:     8443,
		Prefix:         customPrefix,
		StorageBackend: backend,
		EtcdConfig: apiserver.EtcdConfig{
			Servers: etcdServers,
		},
		FileConfig: apiserver.FileConfig{
			RootPath: fileRootpath,
		},
		InsecureAddr:   insecureAddr,
		InsecurePort:   insecurePort,
		Authenticators: nil,
		Authorizers:    nil,
		RequestTimeout: requestTimeout,
		AddToScheme: func(serverScheme, clientScheme *runtime.Scheme) error {
			// register __Internal
			utilruntime.Must(api.AddToScheme(serverScheme))

			// register v1
			utilruntime.Must(apiv1.AddToScheme(serverScheme))
			utilruntime.Must(apiv1.AddToScheme(clientScheme))

			// register v2
			utilruntime.Must(apiv2.AddToScheme(serverScheme))
			utilruntime.Must(apiv2.AddToScheme(clientScheme))
			return nil
		},
		UnrestrictedUpdate:     unrestrictedUpdate,
		OpenAPIDefinitionsFunc: nil,
		Group:                  api.GroupName,
		PrioritizedVersions:    []string{apiv2.SchemeGroupVersion.Version, apiv1.SchemeGroupVersion.Version},
		ResourcesConfig: func(portable bool) api.ResourcesConfig {
			allcfgs := make(map[string]map[string]apiserver.ResourceConfig)
			for version, vcfgs := range api.AllResourcesConfig {
				vcfg := make(map[string]apiserver.ResourceConfig)
				for res, cfg := range vcfgs {
					resmeta := api.ResourceMetadatas[res]
					if portable && resmeta.NoPortable {
						continue
					}
					vcfg[res] = cfg
				}
				allcfgs[version] = vcfg
			}
			return allcfgs
		}(portable),
		AdditionalFilters: nil,
		AdditionalHandlers: map[string]func(clients client.Clients) http.Handler{
			fmt.Sprintf("/openapi/v3"): func(clients client.Clients) http.Handler {
				return openapiHander(insecurePort)
			},
			fmt.Sprintf("%s/", customPrefix): func(clients client.Clients) http.Handler {
				return httputil.NewSingleHostReverseProxy(apiextURL)
			},
		},
	}.Complete()
	if err != nil {
		klog.Fatal(err)
	}

	server := config.New()
	stopCh := genericapiserver.SetupSignalHandler()
	err = server.Prepare(stopCh)
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		klog.Info("Starting up internal shared informers")
		time.Sleep(time.Second)
		err := informer.Start(stopCh)
		if err != nil {
			klog.Fatal(err)
		}
		<-stopCh
		informer.Stop()
	}()

	if err = server.Run(stopCh); err != nil {
		klog.Fatal(err)
	}
}

func openapiHander(insecurePort int) http.Handler {
	return http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			v2 := &openapi2.T{}
			v2url := fmt.Sprintf("http://localhost:%d/openapi/v2", insecurePort)
			resp, err := http.Get(v2url)
			if err != nil {
				respondInternalError(rw, err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				respondInternalError(rw, err)
				return
			}
			v2.UnmarshalJSON(body)
			v3, err := openapi2conv.ToV3(v2)
			if err != nil {
				respondInternalError(rw, err)
				return
			}
			rw.WriteHeader(http.StatusOK)
			yml, err := yaml.Marshal(v3)
			rw.Write(yml)
		},
	)
}

func respondInternalError(rw http.ResponseWriter, err error) {
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(err.Error()))
}
