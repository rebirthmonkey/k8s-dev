package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	//apiexts "cloud.tencent.com/teleport/pkg/apiexts/client"
)

type KubeClientFactory func() client.Client

type Clients interface {
	KubeClient() client.Client
	NoCacheClient() client.Client
	//ApiextsClient() apiexts.Interface
	RestClient() *RestClient
	WithToken(token string) Clients
}

var _ Clients = &clients{}

// RestClient RESTful HTTP client for Teleport API server
type RestClient struct {
	// BaseUrl of Teleport API server
	BaseUrl string
	// HttpClient to be used to send HTTP request
	HttpClient *http.Client
	Token      string

	err error
}

// Put send PUT request
func (c *RestClient) Put(path, staticToken string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var req *http.Request
	url := c.getUrl(path)
	req, err = http.NewRequest(http.MethodPut, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	c.addHeaders(req, staticToken)
	return c.HttpClient.Do(req)
}

func (c *RestClient) WithError(err error) *RestClient {
	n := *c
	n.err = err
	return &n
}

// Get send GET request
func (c *RestClient) Get(path, staticToken string) (*http.Response, error) {
	url := c.getUrl(path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req, staticToken)
	return c.HttpClient.Do(req)
}

func (c *RestClient) do(req *http.Request) (*http.Response, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.HttpClient.Do(req)
}

func (c *RestClient) getUrl(path string) string {
	return fmt.Sprintf("%s%s", c.BaseUrl, path)
}

func (c *RestClient) addHeaders(req *http.Request, staticToken string) {
	req.Header.Add("Content-Type", "application/json")
	token := c.Token
	if token == "" {
		token = staticToken
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
}

// Clients Teleport clients aggregation
type clients struct {
	token        string
	apiServerUrl string
	apiextsUrl   string
	scheme       *runtime.Scheme
	useToken     bool
	lock         sync.RWMutex

	client        client.Client
	nocacheClient client.Client
	cacher        cache.Cache
}

func NewClients(scheme *runtime.Scheme, apiServerUrl, apiextsUrl, staticToken string) *clients {
	return &clients{
		token:        staticToken,
		apiServerUrl: apiServerUrl,
		apiextsUrl:   apiextsUrl,
		scheme:       scheme,
	}
}

// WithToken create a new clients with new token
// TODO implement a client.Client which uses token to make this func work
func (cli *clients) WithToken(token string) Clients {
	if !cli.useToken {
		return cli
	}
	if token == cli.token {
		return cli
	}
	ret := *cli
	ret.token = token
	ret.nocacheClient = nil
	ret.client = nil
	return &ret
}

func (cli *clients) KubeClient() client.Client {
	return cli.NoCacheClient()
	if cli.client != nil {
		return cli.client
	}
	c, err := cli.kubeClient()
	if err != nil {
		//return &errClient{scheme: cli.scheme, err: err}
		return nil
	}
	return c
}

func (cli *clients) NoCacheClient() client.Client {
	if cli.nocacheClient != nil {
		cli.lock.RLock()
		defer cli.lock.RUnlock()
		return cli.nocacheClient
	}
	c, err := cli.noCacheClient()
	if err != nil {
		//return &errClient{scheme: cli.scheme, err: err}
		return nil
	}
	cli.lock.Lock()
	cli.nocacheClient = c
	cli.lock.Unlock()
	return c
}

func (cli *clients) RestClient() *RestClient {
	c, err := cli.restConfig()
	if err != nil {
		rc := &RestClient{BaseUrl: "", Token: cli.token}
		return rc.WithError(err)
	}
	return &RestClient{BaseUrl: c.Host, Token: cli.token}
}

//func (cli *clients) ApiextsClient() apiexts.Interface {
//	c, err := cli.restConfig()
//	if err != nil {
//		return apiexts.NewClient("", cli.token).WithError(fmt.Errorf("can't find api server url: %w", err))
//	}
//	return apiexts.NewClient(c.Host, cli.token)
//}

func (cli *clients) kubeClient() (client.Client, error) {
	c, err := cli.restConfig()
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(c)
	if err != nil {
		return nil, err
	}
	cacher, err := cli.getCacher(c, mapper)
	if err != nil {
		return nil, err
	}
	return cluster.DefaultNewClient(cacher, c, client.Options{Scheme: cli.scheme, Mapper: mapper})
}

func (cli *clients) getCacher(c *rest.Config, mapper meta.RESTMapper) (cacher cache.Cache, err error) {
	if cli.cacher != nil {
		return cli.cacher, nil
	}
	cli.cacher, err = cache.New(c, cache.Options{
		Scheme: cli.scheme,
		Mapper: mapper,
	})
	if err == nil {
		go func() {
			err := cli.cacher.Start(context.Background())
			if err != nil {
				// handle error
			}
		}()
	}

	return cli.cacher, err
}

func (cli *clients) noCacheClient() (client.Client, error) {
	c, err := cli.restConfig()
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(c)
	if err != nil {
		return nil, err
	}
	return client.New(c, client.Options{Scheme: cli.scheme, Mapper: mapper})

}

func (cli *clients) restConfig() (c *rest.Config, err error) {
	func() {
		if cli.apiServerUrl == "" {
			c, err = config.GetConfig()
			return
		}
		c, err = clientcmd.BuildConfigFromFlags(cli.apiServerUrl, "")
	}()
	if err == nil && cli.useToken {
		// this is useless
		c.BearerToken = cli.token
	}
	return
}
