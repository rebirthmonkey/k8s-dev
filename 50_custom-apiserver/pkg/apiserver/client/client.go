package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeClientFactory func() client.Client
type RestClientFactory func() RestClient

// RestClient RESTful HTTP client for Teleport API server
type RestClient struct {
	// BaseUrl of Teleport API server
	BaseUrl string
	// HttpClient to be used to send HTTP request
	HttpClient *http.Client
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

func (c *RestClient) getUrl(path string) string {
	return fmt.Sprintf("%s%s", c.BaseUrl, path)
}

func (c *RestClient) addHeaders(req *http.Request, staticToken string) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", staticToken))
}

// Clients Teleport clients aggregation
type Clients struct {
	// create a controller-runtime client with strong type support
	KubeClient KubeClientFactory
	// create a http client
	RestClient RestClientFactory
}
