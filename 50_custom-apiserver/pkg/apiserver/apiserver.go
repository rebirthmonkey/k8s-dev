package apiserver

import (
	"50_custom-apiserver/pkg/apiserver/filters"
	"50_custom-apiserver/pkg/apiserver/filters/reqinfo"
	"50_custom-apiserver/pkg/apiserver/store/etcd"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"50_custom-apiserver/pkg/apiserver/consts"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	unionauthz "k8s.io/apiserver/pkg/authorization/union"

	"50_custom-apiserver/pkg/apiserver/client"
	"50_custom-apiserver/pkg/apiserver/names"
	"50_custom-apiserver/pkg/apiserver/store"
	"50_custom-apiserver/pkg/apiserver/strategy"
	"50_custom-apiserver/pkg/apiserver/tabconv"
	"50_custom-apiserver/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	unionauthn "k8s.io/apiserver/pkg/authentication/request/union"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kube-openapi/pkg/common"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// TeleportServer apiserver operations
type TeleportServer interface {
	// Prepare required components
	Prepare(stopCh <-chan struct{}) error
	// Run the server
	Run(stopCh <-chan struct{}) error
}

// Config apiserver configuration
type Config struct {
	Name                   string
	Prefix                 string
	SecureAddr             string
	SecurePort             int
	InsecureAddr           string
	InsecurePort           int
	Group                  string
	AddToScheme            func(serverScheme, clientScheme *runtime.Scheme) error
	CertDirectory          string
	ResourcesConfig        map[string]map[string]ResourceConfig
	PostStartHooks         map[string]genericapiserver.PostStartHookFunc
	AdditionalHandlers     map[string]func(clients client.Clients) http.Handler
	StorageBackend         string
	EtcdConfig             EtcdConfig
	FileConfig             FileConfig
	RequestTimeout         time.Duration
	UnrestrictedUpdate     bool
	OpenAPIDefinitionsFunc common.GetOpenAPIDefinitions
	PrioritizedVersions    []string
	Authenticators         []authenticator.Request
	Authorizers            []authorizer.Authorizer
	AdditionalFilters      func(wrapped http.Handler, s runtime.NegotiatedSerializer) (wrapper http.Handler)
}

// EtcdConfig config for etcd backend
type EtcdConfig struct {
	Prefix  string
	Servers []string
}

// FileConfig config for file backend
type FileConfig struct {
	RootPath string
}

// ResourceConfig config for specific resource
type ResourceConfig struct {
	// callback to create a new resource
	NewFunc func() runtime.Object
	// callback to create a new resource list
	NewListFunc func() runtime.Object
	// hook on PrepareForCreate
	PrepareForCreateFunc strategy.PrepareForCreateFunc
	// hook on Validate
	ValidateFunc strategy.ValidateFunc
	// hook on PrepareForUpdate
	PrepareForUpdateFunc strategy.PrepareForUpdateFunc
	// hook on ValidateUpdate
	ValidateUpdateFunc strategy.ValidateUpdateFunc
	// hook on Canonicalize
	CanonicalizeFunc strategy.CanonicalizeFunc
	// to create a TableConvertor
	TableConvertorBuilder tabconv.TableConvertorBuilder
	// configurations for sub-resources
	SubResourcesConfig map[string]SubResourceConfig
	ShortNames         []string
}

// SubResourceConfig config for a subresource
type SubResourceConfig struct {
	// hook on PrepareForUpdate
	PrepareForUpdateFunc strategy.PrepareForUpdateFunc
	// hook on ValidateUpdate
	ValidateUpdateFunc strategy.ValidateUpdateFunc
	// hook on Canonicalize
	CanonicalizeFunc strategy.CanonicalizeFunc
}

type completedConfig struct {
	config *Config
}

// Complete apply defaults to the config
func (c Config) Complete() (completedConfig, error) {
	if c.Name == "" {
		c.Name = "teleport-apiserver"
	}
	if c.StorageBackend == consts.StorageBackendEtcd {
		if c.EtcdConfig.Prefix == "" {
			c.EtcdConfig.Prefix = c.Prefix
		}
		if len(c.EtcdConfig.Servers) == 0 {
			return completedConfig{}, errors.New("empty etcd servers")
		}
	} else if c.StorageBackend == consts.StorageBackendFile {
		if c.FileConfig.RootPath == "" {
			return completedConfig{}, errors.New("empty storage root path")
		}
	}
	if c.CertDirectory == "" {
		c.CertDirectory = "/tmp/teleport/certs"
	}
	localhost := utils.LocalhostIP
	if c.SecureAddr == "" {
		c.SecureAddr = localhost
	}
	if c.InsecureAddr == "" {
		c.InsecureAddr = localhost
	}
	if c.SecurePort == 0 {
		c.SecurePort = 6443
	}
	if c.InsecurePort == 0 {
		c.InsecurePort = 6080
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = time.Second * 15
	}
	return completedConfig{&c}, nil
}

// New create a new apiserver
func (c *completedConfig) New() TeleportServer {
	ts := &teleportServer{config: c.config}
	ts.scheme = runtime.NewScheme()
	ts.clientScheme = runtime.NewScheme()
	ts.codecFactory = serializer.NewCodecFactory(ts.scheme)
	return ts
}

type teleportServer struct {
	config                 *Config
	scheme                 *runtime.Scheme
	clientScheme           *runtime.Scheme
	codecFactory           serializer.CodecFactory
	completedGenericConfig genericapiserver.CompletedConfig
	genericServer          *genericapiserver.GenericAPIServer
	clients                client.Clients
}

func (t *teleportServer) Run(stopCh <-chan struct{}) (err error) {
	preparedServer := t.genericServer.PrepareRun()
	httpServer := http.Server{
		Addr:    fmt.Sprintf("%s:%d", t.config.InsecureAddr, t.config.InsecurePort),
		Handler: preparedServer.Handler,
	}
	go func() {
		preparedServer.RunPostStartHooks(stopCh)
		<-stopCh
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		preparedServer.RunPreShutdownHooks()
		err = httpServer.Shutdown(ctx)
	}()
	httpServer.ListenAndServe()
	return err
}
func storeInPeace(storage rest.StandardStorage, err error) rest.StandardStorage {
	if err != nil {
		panic(err)
	}
	return storage
}
func (t *teleportServer) createStore(gr schema.GroupResource, newFunc, newListFunc func() runtime.Object,
	tcf tabconv.TableConvertorBuilder, stragety strategy.Strategy, parent rest.StandardStorage, shortNames []string) (
	rest.StandardStorage, error) {
	tc, err := tcf(gr)
	if err != nil {
		return nil, err
	}
	attrs := func(obj runtime.Object) (labels.Set, fields.Set, error) {
		typ := reflect.TypeOf(newFunc())
		if reflect.TypeOf(obj) != typ {
			return nil, nil, fmt.Errorf("given object is not a %s", typ.Name())
		}
		oma := obj.(metav1.ObjectMetaAccessor)
		meta := oma.GetObjectMeta()
		return meta.GetLabels(), fields.Set{
			"metadata.name":      meta.GetName(),
			"metadata.namespace": meta.GetNamespace(),
		}, nil
	}
	switch t.config.StorageBackend {
	case consts.StorageBackendEtcd:
		es := &genericregistry.Store{
			NewFunc:     newFunc,
			NewListFunc: newListFunc,
			PredicateFunc: func(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
				return storage.SelectionPredicate{
					Label:    label,
					Field:    field,
					GetAttrs: attrs,
				}
			},
			DefaultQualifiedResource: gr,

			CreateStrategy: stragety,
			UpdateStrategy: stragety,
			DeleteStrategy: stragety,

			TableConvertor: tc,
		}
		options := &generic.StoreOptions{RESTOptions: t.completedGenericConfig.RESTOptionsGetter, AttrFunc: attrs}
		if err := es.CompleteWithOptions(options); err != nil {
			return nil, err
		}
		if parent == nil && utils.IsNotEmpty(shortNames) {
			return etcd.WithShortNames(es, shortNames), nil
		}
		return es, nil
	//case consts.StorageBackendFile:
	//	s := t.scheme
	//	codec, _, err := serverstorage.NewStorageCodec(serverstorage.StorageCodecConfig{
	//		StorageMediaType:  runtime.ContentTypeYAML,
	//		StorageSerializer: serializer.NewCodecFactory(s),
	//		StorageVersion:    s.PrioritizedVersionsForGroup(gr.Group)[0],
	//		MemoryVersion:     s.PrioritizedVersionsForGroup(gr.Group)[0],
	//		Config:            storagebackend.Config{},
	//	})
	//	if err != nil {
	//		return nil, err
	//	}
	//	var parentStore *file.Store
	//	if parent != nil {
	//		switch p := parent.(type) {
	//		case *file.ShortNamesProvider:
	//			parentStore = p.Store
	//		case *file.Store:
	//			parentStore = p
	//		}
	//	}
	//	fs := file.NewStore(
	//		parentStore,
	//		gr,
	//		codec,
	//		t.config.FileConfig.RootPath,
	//		true,
	//		newFunc,
	//		newListFunc,
	//		tc,
	//		stragety,
	//	)
	//	if parent == nil && utils.IsNotEmpty(shortNames) {
	//		return file.WithShortNames(fs, shortNames), nil
	//	}
	//	return fs, nil
	default:
		return nil, errors.New(fmt.Sprintf("unkown storage backend %s", t.config.StorageBackend))
	}
}

// Prepare required components
func (t *teleportServer) Prepare(stopCh <-chan struct{}) error {
	serverScheme := t.scheme
	clientScheme := t.clientScheme

	unversioned := schema.GroupVersion{Group: "", Version: "v1"}

	// metav1 types should add to unversioned & each api groupversion ( except __internal version )
	metav1.AddToGroupVersion(serverScheme, unversioned)

	serverScheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
	cfg := t.config
	group := cfg.Group
	var prioritizedVersions []schema.GroupVersion
	for _, v := range cfg.PrioritizedVersions {
		prioritizedVersions = append(prioritizedVersions, schema.GroupVersion{
			Group:   group,
			Version: v,
		})
	}
	// register all api versions to server scheme & client scheme
	cfg.AddToScheme(serverScheme, clientScheme)
	utilruntime.Must(serverScheme.SetVersionPriority(prioritizedVersions...))
	utilruntime.Must(clientScheme.SetVersionPriority(prioritizedVersions...))

	storageGroupVersion := prioritizedVersions[0]
	codecFactory := t.codecFactory
	// LegacyCodec encodes output to a given API versions, and decodes output into the internal form from
	// any recognized source. The returned codec will always encode output to JSON. If a type is not
	// found in the list of versions an error will be returned.
	codec := codecFactory.LegacyCodec(prioritizedVersions...)
	// genericOptions.Etcd will use this codec
	genericOptions := genericoptions.NewRecommendedOptions(
		cfg.Prefix,
		codec,
	)

	genericOptions.SecureServing.ServerCert.CertDirectory = cfg.CertDirectory
	ips := []net.IP{net.ParseIP(utils.LocalhostIP)}
	if err := genericOptions.SecureServing.MaybeDefaultWithSelfSignedCerts(utils.Localhost, nil, ips); err != nil {
		return err
	}

	if cfg.StorageBackend == consts.StorageBackendEtcd {
		// we have single group only
		genericOptions.Etcd.StorageConfig.EncodeVersioner =
			runtime.NewMultiGroupVersioner(storageGroupVersion, schema.GroupKind{Group: storageGroupVersion.Group})
		genericOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)
		genericOptions.Etcd.StorageConfig.Transport.ServerList = cfg.EtcdConfig.Servers
	} else {
		genericOptions.Etcd = nil
	}
	genericOptions.Authentication = nil
	genericOptions.Authorization = nil
	genericOptions.CoreAPI = nil
	genericOptions.Admission = nil
	genericOptions.SecureServing.BindAddress = net.ParseIP(cfg.SecureAddr)
	genericOptions.SecureServing.BindPort = cfg.SecurePort

	genericConfig := genericapiserver.NewRecommendedConfig(codecFactory)
	genericConfig.RequestTimeout = cfg.RequestTimeout
	genericConfig.BuildHandlerChainFunc = func(handler http.Handler, c *genericapiserver.Config) http.Handler {
		if cfg.AdditionalFilters != nil {
			handler = cfg.AdditionalFilters(handler, c.Serializer)
		}
		handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.LongRunningFunc)
		handler = filters.WithAuthorization(handler, c.Authorization.Authorizer, c.Serializer)
		handler = genericapifilters.WithAuthentication(handler, c.Authentication.Authenticator,
			genericapifilters.Unauthorized(c.Serializer), c.Authentication.APIAudiences)
		handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)
		handler = genericapifilters.WithRequestInfo(handler, reqinfo.NewRequestInfoResolver(cfg.Prefix))
		handler = genericapifilters.WithRequestInfo(handler, c.RequestInfoResolver)
		handler = genericapifilters.WithCacheControl(handler)
		handler = genericapifilters.WithRequestReceivedTimestamp(handler)
		handler = genericfilters.WithHTTPLogging(handler)
		handler = genericfilters.WithPanicRecovery(handler, c.RequestInfoResolver)
		return handler
	}
	if cfg.OpenAPIDefinitionsFunc != nil {
		genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(cfg.OpenAPIDefinitionsFunc,
			openapi.NewDefinitionNamer(serverScheme))
		genericConfig.OpenAPIConfig.Info.Title = "Teleport"
		genericConfig.OpenAPIConfig.Info.Version = "1.0"
	}

	utilfeature.DefaultMutableFeatureGate.SetFromMap(map[string]bool{
		string(features.APIPriorityAndFairness): false,
		string(features.ServerSideApply):        false,
	})

	if err := genericOptions.ApplyTo(genericConfig); err != nil {
		return err
	}

	insecureUrl := fmt.Sprintf("http://%s:%d", utils.LocalhostIP, t.config.InsecurePort)
	insecureCfg, err := clientcmd.BuildConfigFromFlags(insecureUrl, "")
	insecureCfg.Insecure = true
	insecureCfg.BearerToken = "BpLnfgDsc2WD8F2qNfHK5a84"
	if err != nil {
		return err
	}
	genericConfig.LoopbackClientConfig = insecureCfg

	genericConfig.Authorization = genericapiserver.AuthorizationInfo{
		Authorizer: unionauthz.New(cfg.Authorizers...),
	}
	genericConfig.Authentication = genericapiserver.AuthenticationInfo{
		Authenticator: unionauthn.NewFailOnError(cfg.Authenticators...),
	}

	completedConfig := genericConfig.Complete()
	t.completedGenericConfig = completedConfig
	genericServer, err := completedConfig.New(cfg.Name, genericapiserver.NewEmptyDelegate())
	if err != nil {
		return err
	}
	t.genericServer = genericServer
	kubeClientFactory := func() ctrlrtclient.Client {
		mapper, _ := apiutil.NewDiscoveryRESTMapper(genericConfig.LoopbackClientConfig)
		cli, _ := ctrlrtclient.New(genericConfig.LoopbackClientConfig, ctrlrtclient.Options{
			Scheme: clientScheme,
			Mapper: mapper,
		})
		return cli
	}
	t.clients = client.Clients{
		KubeClient: kubeClientFactory,
		RestClient: func() client.RestClient {
			return client.RestClient{
				BaseUrl: fmt.Sprintf("http://127.0.0.1:%d", cfg.InsecurePort),
				HttpClient: &http.Client{
					Transport: &http.Transport{
						MaxIdleConns:       10,
						IdleConnTimeout:    30 * time.Second,
						DisableCompression: true,
					},
					Timeout: cfg.RequestTimeout,
				},
			}
		},
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(group, serverScheme, metav1.ParameterCodec, codecFactory)

	apiGroupInfo.PrioritizedVersions = prioritizedVersions

	for version, resourceConfigs := range cfg.ResourcesConfig {
		versionStorage := map[string]rest.Storage{}
		apiGroupInfo.VersionedResourcesStorageMap[version] = versionStorage
		for resource, resourceConfig := range resourceConfigs {
			tcb := resourceConfig.TableConvertorBuilder
			if tcb == nil {
				tcb = func(gr schema.GroupResource) (rest.TableConvertor, error) {
					return rest.NewDefaultTableConvertor(gr), nil
				}
			}
			if err != nil {
				return err
			}
			stragety := strategy.DefaultStrategy{
				ObjectTyper:          serverScheme,
				NameGenerator:        names.DefaultNameGenerator,
				PrepareForCreateFunc: resourceConfig.PrepareForCreateFunc,
				ValidateFunc:         resourceConfig.ValidateFunc,
				PrepareForUpdateFunc: resourceConfig.PrepareForUpdateFunc,
				ValidateUpdateFunc:   resourceConfig.ValidateUpdateFunc,
				CanonicalizeFunc:     resourceConfig.CanonicalizeFunc,
				ClientFactory:        kubeClientFactory,
				Unrestricted:         cfg.UnrestrictedUpdate,
				Resource:             resource,
			}
			parentgr := schema.GroupResource{Group: group, Resource: resource}
			parentStore := storeInPeace(t.createStore(
				parentgr,
				resourceConfig.NewFunc,
				resourceConfig.NewListFunc,
				tcb,
				stragety, nil, resourceConfig.ShortNames,
			))
			versionStorage[resource] = parentStore
			for subres, subcfg := range resourceConfig.SubResourcesConfig {
				subgr := schema.GroupResource{Group: group, Resource: fmt.Sprintf("%s/%s", resource, subres)}
				versionStorage[subgr.Resource] = store.PartialUpdateStore(
					storeInPeace(t.createStore(
						parentgr,
						resourceConfig.NewFunc,
						resourceConfig.NewListFunc,
						tcb,
						stragety,
						parentStore, nil,
					)), subcfg.PrepareForUpdateFunc, subcfg.ValidateUpdateFunc, subcfg.CanonicalizeFunc,
					stragety.ClientFactory)
			}
		}

	}

	if err := genericServer.InstallAPIGroups(&apiGroupInfo); err != nil {
		return err
	}

	for name, hook := range cfg.PostStartHooks {
		genericServer.AddPostStartHookOrDie(name, hook)
	}

	for ptn, hdl := range cfg.AdditionalHandlers {
		mux := genericServer.Handler.NonGoRestfulMux
		switch {
		case strings.HasSuffix(ptn, "/"):
			mux.HandlePrefix(ptn, hdl(t.clients))
		default:
			mux.Handle(ptn, hdl(t.clients))
		}
	}

	return nil
}
