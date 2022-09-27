package authn

import (
	"net/http"
	"strings"

	"50_custom-apiserver/pkg/apiserver/consts"
	"50_custom-apiserver/pkg/utils"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

type allowLocalHost struct {
	authenticator authenticator.Request
	defaultUser   user.Info
}

// AuthenticateRequest implements authenticator.Request.
func (p allowLocalHost) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	realIp := getRealIP(req)
	if strings.Index(realIp, utils.LocalhostIP) == 0 {
		return &authenticator.Response{User: p.defaultUser}, true, nil
	}
	return nil, false, nil
}

func NewAllowLocalHost() authenticator.Request {
	pla := allowLocalHost{
		defaultUser: &user.DefaultInfo{
			Name:   consts.SuperUserName,
			UID:    consts.SuperUserUID,
			Groups: nil,
			Extra:  nil,
		},
	}
	return pla
}

// getRealIP to get real IP from proxy
func getRealIP(req *http.Request) string {
	if ip := req.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return req.RemoteAddr
}
