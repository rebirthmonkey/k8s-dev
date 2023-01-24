package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Resp struct {
	Code int
	Data string
}

func (s Resp) JsonBody(to interface{}) error {
	return s.JsonUnmarshal(to)
}

func (s Resp) IsNotFound() bool {
	return s.Code == 404
}

func (s Resp) Succeeded() bool {
	return s.Code >= 200 && s.Code < 300
}

// NOTE use JsonBody instead, this method will be abandoned in future version
func (s Resp) JsonUnmarshal(to interface{}) error {
	if err := s.AsError(); err != nil {
		return err
	}
	return json.Unmarshal([]byte(s.Data), to)
}

func (s Resp) AsError() error {
	if s.Code >= 200 && s.Code < 300 {
		return nil
	}
	return fmt.Errorf("%d: %s", s.Code, s.Data)
}

type Interface interface {
	Create(kind string, name string, data ...interface{}) Resp
	Update(kind string, name string, data ...interface{}) Resp
	Delete(kind string, name string) Resp
	Get(kind string, name string) Resp
	GetAll(kind string) Resp
	With(sub string) Interface
	WithVersion(v string) Interface
	WithTimeout(timeout time.Duration) Interface
	WithNamespace(ns string) Interface
	WithError(err error) Interface
}

type Client struct {
	url       string
	prefix    string
	token     string
	namespace string
	subrsc    string
	v         string
	err       error
	timeout   time.Duration
}

func statusAny(code int, data string) Resp {
	return Resp{Code: code, Data: data}
}

func (g Client) Create(kind string, name string, data ...interface{}) Resp {
	var req *http.Request
	var err error
	var body io.Reader
	if body, err = dataReader(data...); err != nil {
		return statusAny(400, err.Error())
	}
	url := g.getUrl(name, kind)
	req, err = http.NewRequest("POST", url, body)
	if err != nil {
		return statusAny(400, err.Error())
	}
	status := g.doHttpRequest(req)
	return status
}

func (g Client) Update(kind string, name string, data ...interface{}) Resp {
	var req *http.Request
	var err error
	var body io.Reader
	if body, err = dataReader(data...); err != nil {
		return statusAny(400, err.Error())
	}
	req, err = http.NewRequest("PUT", g.getUrlWithSub(name, kind), body)
	if err != nil {
		return statusAny(500, err.Error())
	}
	status := g.doHttpRequest(req)
	return status
}

func (g Client) Delete(kind string, name string) Resp {
	req, err := http.NewRequest("DELETE", g.getUrl(name, kind), nil)
	if err != nil {
		return statusAny(500, err.Error())
	}
	status := g.doHttpRequest(req)
	return status
}

func (g Client) Get(kind string, name string) Resp {
	req, err := http.NewRequest("GET", g.getUrl(name, kind), nil)
	if err != nil {
		return statusAny(400, err.Error())
	}
	return g.doHttpRequest(req)
}

func (g Client) doHttpRequest(req *http.Request) Resp {
	if g.err != nil {
		return statusAny(500, g.err.Error())
	}
	timeout := g.timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	if g.token != "" {
		token := g.token
		if strings.Index(token, "Bearer") < 0 {
			token = "Bearer " + token
		}
		req.Header.Add("Authorization", token)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return statusAny(500, err.Error())
	}
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return statusAny(500, err.Error())
	}
	return statusAny(resp.StatusCode, string(bs))
}

func (g Client) getUrl(name, kind string) string {
	namespace := g.namespace
	if namespace == "" {
		namespace = "default"
	}
	url := fmt.Sprintf("%s%s/%s/%s/%s", g.url, g.prefix, kind, g.v, namespace)
	if name != "" {
		url += "/" + name
	}
	return url
}

func (g Client) getUrlWithSub(name, kind string) string {
	url := g.getUrl(name, kind)
	if g.subrsc != "" {
		s := g.subrsc
		if s[0] != '/' {
			s = "/" + s
		}
		url += s
		g.subrsc = ""
	}
	return url
}

func (g Client) GetAll(kind string) Resp {
	url := func() string {
		namespace := g.namespace
		if namespace == "" {
			namespace = "default"
		}
		return fmt.Sprintf("%s%s/%s/%s/%s", g.url, g.prefix, kind, g.v, namespace)
	}()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return statusAny(400, err.Error())
	}
	return g.doHttpRequest(req)
}

func (g *Client) With(sub string) Interface {
	n := *g
	n.subrsc = sub
	return &n
}

func (g *Client) WithVersion(v string) Interface {
	n := *g
	n.v = v
	return &n
}

func (g *Client) WithTimeout(timeout time.Duration) Interface {
	n := *g
	n.timeout = timeout
	return &n
}

func (g *Client) WithNamespace(ns string) Interface {
	n := *g
	n.namespace = ns
	return &n
}

func (g *Client) WithError(err error) Interface {
	n := *g
	n.err = err
	return &n
}

// NewClient create a new generic apiext client
func NewClient(url, token string) *Client {
	return &Client{
		url:    url,
		prefix: "/teleport",
		token:  token,
		v:      "v1",
	}
}

func NewNamespacedClient(url, token, namespace string) *Client {
	return &Client{
		url:       url,
		prefix:    "/teleport",
		token:     token,
		namespace: namespace,
		v:         "v1",
	}
}

func dataReader(data ...interface{}) (r io.Reader, err error) {
	var body string
	if len(data) > 0 {
		var bs []byte
		bs, err = json.Marshal(data[0])
		if err != nil {
			return
		}
		body = string(bs)
	}
	r = strings.NewReader(body)
	return
}
