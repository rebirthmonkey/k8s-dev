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
)

type KubeClientFactory func() client.Client

type Clients interface {
	KubeClient() client.Client
	NoCacheClient() client.Client
	ApiextsClient() Interface
	RestClient() *RestClient
	WithToken(token string) Clients
}

var _ Clients = &clients{}

type clients struct {
	token         string
	lock          sync.RWMutex
	client        client.Client
	nocacheClient client.Client
	manager       *clientsManager
	cacher        cache.Cache
}

// Clients Teleport clients aggregation
type clientsManager struct {
	apiServerUrl string
	apiextsUrl   string
	scheme       *runtime.Scheme
	useToken     bool
	lock         sync.RWMutex
	clients      map[string]*clients
	global       *clients
}

func newClients(manager *clientsManager) *clients {
	return &clients{
		manager: manager,
	}
}

func NewClientsManager(scheme *runtime.Scheme, apiServerUrl, apiextsUrl string) *clientsManager {
	return &clientsManager{
		apiServerUrl: apiServerUrl,
		apiextsUrl:   apiextsUrl,
		scheme:       scheme,
	}
}

// WithToken create a new clients with new token
// TODO implement a client.Client which uses token to make this func work
func (cli *clientsManager) WithToken(token string) Clients {
	if !cli.useToken {
		return cli.getGlobal()
	}
	return cli.getTokenClients(token)
}

func (cli *clientsManager) getGlobal() Clients {
	cli.lock.RLock()
	global := cli.global
	cli.lock.RUnlock()
	if global != nil {
		return global
	}
	cli.lock.Lock()
	defer cli.lock.Unlock()
	cli.global = newClients(cli)
	return cli.global
}

func (cli *clientsManager) getTokenClients(token string) Clients {
	cli.lock.RLock()
	clients, ok := cli.clients[token]
	cli.lock.RUnlock()
	if ok {
		return clients
	}
	cli.lock.Lock()
	cli.clients[token] = newClients(cli)
	cli.lock.Unlock()
	return cli.clients[token]
}

func (cli *clientsManager) KubeClient() client.Client {
	return cli.getGlobal().KubeClient()
}

func (cli *clientsManager) NoCacheClient() client.Client {
	return cli.getGlobal().NoCacheClient()
}

func (cli *clientsManager) ApiextsClient() Interface {
	return cli.getGlobal().ApiextsClient()
}

func (cli *clientsManager) RestClient() *RestClient {
	if cli.global == nil {
		cli.global = newClients(cli)
	}
	return cli.global.RestClient()
}

func (cli *clients) WithToken(token string) Clients {
	cli.token = token
	return cli
}

func (cli *clients) KubeClient() client.Client {
	cli.lock.RLock()
	kube := cli.client
	cli.lock.RUnlock()
	if kube != nil {
		return kube
	}
	c, err := cli.kubeClient()
	if err != nil {
		//return &errClient{scheme: cli.manager.scheme, err: err}
		return nil
	}
	cli.lock.Lock()
	cli.client = c
	cli.lock.Unlock()
	return c
}

func (cli *clients) NoCacheClient() client.Client {
	cli.lock.RLock()
	nocache := cli.nocacheClient
	cli.lock.RUnlock()
	if nocache != nil {
		return nocache
	}
	var err error
	nocache, err = cli.noCacheClient()
	if err != nil {
		//return &errClient{scheme: cli.manager.scheme, err: err}
		return nil
	}
	cli.lock.Lock()
	cli.nocacheClient = nocache
	cli.lock.Unlock()
	return nocache
}

func (cli *clients) RestClient() *RestClient {
	c, err := cli.restConfig()
	if err != nil {
		rc := &RestClient{BaseUrl: "", Token: cli.token}
		return rc.WithError(err)
	}
	return &RestClient{BaseUrl: c.Host, Token: cli.token}
}

func (cli *clients) restConfig() (*rest.Config, error) {
	c, err := cli.manager.restConfig()
	if err != nil {
		return nil, err
	}
	if cli.manager.useToken {
		c.BearerToken = cli.token
	}
	return c, nil
}

func (cli *clients) ApiextsClient() Interface {
	return NewClient(cli.manager.apiextsUrl, cli.token)
}

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
	return cluster.DefaultNewClient(cacher, c, client.Options{Scheme: cli.manager.scheme, Mapper: mapper})
}

func (cli *clients) getCacher(c *rest.Config, mapper meta.RESTMapper) (cacher cache.Cache, err error) {
	if cli.cacher != nil {
		return cli.cacher, nil
	}

	cli.cacher, err = cache.New(c, cache.Options{
		Scheme: cli.manager.scheme,
		Mapper: mapper,
	})
	if err == nil {
		go func() {
			for i := 0; i < 10; i++ {
				err := cli.cacher.Start(context.Background())
				if err != nil {
					// handle error
				}
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
	return client.New(c, client.Options{Scheme: cli.manager.scheme, Mapper: mapper})

}

func (cli *clientsManager) restConfig() (c *rest.Config, err error) {
	func() {
		if cli.apiServerUrl == "" {
			c, err = config.GetConfig()
			return
		}
		c, err = clientcmd.BuildConfigFromFlags(cli.apiServerUrl, "")
	}()
	return
}

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
