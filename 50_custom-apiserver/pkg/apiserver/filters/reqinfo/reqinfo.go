package reqinfo

import (
	"net/http"
	"strings"

	"k8s.io/apiserver/pkg/endpoints/request"
)

func NewRequestInfoResolver(prefix string) request.RequestInfoResolver {
	return &requestInfoResolver{
		prefix: prefix,
	}
}

type requestInfoResolver struct {
	prefix string
}

func (r *requestInfoResolver) NewRequestInfo(req *http.Request) (*request.RequestInfo, error) {
	requestInfo, ok := request.RequestInfoFrom(req.Context())
	if !ok {
		requestInfo = &request.RequestInfo{
			IsResourceRequest: false,
			Path:              req.URL.Path,
			Verb:              strings.ToLower(req.Method),
		}
	}
	if strings.Index(requestInfo.Path, r.prefix) == 0 {
		currentParts := splitPath(requestInfo.Path)
		if len(currentParts) >= 4 {
			requestInfo.Resource = currentParts[1]
			requestInfo.APIVersion = currentParts[2]
			requestInfo.Namespace = currentParts[3]
			if len(currentParts) >= 5 {
				requestInfo.Name = currentParts[4]
			}
		}
	}
	return requestInfo, nil
}

// splitPath returns the segments for a URL path.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
