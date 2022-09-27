package authn

import (
	"50_custom-apiserver/pkg/apiserver/consts"
	"errors"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"net/http"
	"strings"
)

type kubectl struct {
	kubectlDisabled bool
}

func (k *kubectl) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	ua := req.UserAgent()
	if strings.Index(strings.ToLower(ua), consts.Kubectl) == 0 {
		if k.kubectlDisabled {
			return nil, false, errors.New("kubectl disabled")
		}
	}
	return nil, false, nil
}

func NewKubectl(kubectlDisabled bool) authenticator.Request {
	return &kubectl{
		kubectlDisabled: kubectlDisabled,
	}
}
