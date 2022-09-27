package filters

import (
	"50_custom-apiserver/pkg/apiserver/consts"
	"errors"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
)

const (
	// Annotation key names set in advanced audit
	decisionAnnotationKey = "authorization.k8s.io/decision"
	reasonAnnotationKey   = "authorization.k8s.io/reason"

	// Annotation values set in advanced audit
	decisionAllow  = "allow"
	decisionForbid = "forbid"
	reasonError    = "internal error"
)

func WithAuthorization(handler http.Handler, a authorizer.Authorizer, s runtime.NegotiatedSerializer) http.Handler {
	if a == nil {
		klog.Warning("Authorization is disabled")
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		attributes, err := GetAuthorizerAttributes(req)
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}
		authorized, reason, err := a.Authorize(ctx, attributes)
		if authorized == authorizer.DecisionAllow {
			handler.ServeHTTP(w, req)
			return
		}
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}

		klog.V(4).InfoS("Forbidden", "URI", req.RequestURI, "Reason", reason)
		responsewriters.Forbidden(ctx, attributes, w, req, reason, s)
	})
}

func GetAuthorizerAttributes(req *http.Request) (authorizer.Attributes, error) {
	ctx := req.Context()
	attribs := authorizer.AttributesRecord{}

	u, ok := request.UserFrom(ctx)
	if ok {
		attribs.User = u
		ui, ok := u.(*user.DefaultInfo)
		if ok {
			ua := req.UserAgent()
			if strings.Index(strings.ToLower(ua), consts.Kubectl) == 0 {
				if ui.Extra == nil {
					ui.Extra = make(map[string][]string)
				}
				ui.Extra[consts.Kubectl] = []string{}
			}
		}
	}

	requestInfo, found := request.RequestInfoFrom(ctx)
	if !found {
		return nil, errors.New("no RequestInfo found in the context")
	}

	// Start with common attributes that apply to resource and non-resource requests
	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Verb = requestInfo.Verb

	attribs.APIGroup = requestInfo.APIGroup
	attribs.APIVersion = requestInfo.APIVersion
	attribs.Resource = requestInfo.Resource
	attribs.Subresource = requestInfo.Subresource
	attribs.Namespace = requestInfo.Namespace
	attribs.Name = requestInfo.Name

	return &attribs, nil
}
