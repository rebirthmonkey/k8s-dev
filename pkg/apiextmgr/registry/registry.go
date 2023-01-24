package registry

import (
	"github.com/gin-gonic/gin"
	ginserver "github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
)

var apiExtHandlerMgr APIExtHandlerManager

func init() {
	apiExtHandlerMgr = APIExtHandlerManager{[]APIExtHandler{}}
}

// APIExtHandler apiext handler register entry
type APIExtHandler struct {
	RegisterFunc func(string, *gin.Engine)
	Prefix       string
}

type APIExtHandlerManager struct {
	scheme []APIExtHandler
}

func (m *APIExtHandlerManager) Register(handler APIExtHandler) {
	m.scheme = append(m.scheme, handler)
}

func (m *APIExtHandlerManager) AddToManager(ginEngine *gin.Engine) {
	for _, handler := range m.scheme {
		log.Infof("[APIExtHandlerManager] Register Handler for %s", handler.Prefix)
		handler.RegisterFunc(handler.Prefix, ginEngine)
	}
}

func Register(handler APIExtHandler) {
	apiExtHandlerMgr.Register(handler)
}

func AddToManager(ginServer *ginserver.PreparedServer) {
	apiExtHandlerMgr.AddToManager(ginServer.Engine)
}
