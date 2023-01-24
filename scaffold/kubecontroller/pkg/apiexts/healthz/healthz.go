package healthz

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr/registry"
	"github.com/rebirthmonkey/k8s-dev/pkg/k8s/client"
)

func init() {
	registry.Register(registry.APIExtHandler{
		RegisterFunc: RegisterFunc,
		Prefix:       "/v1",
	})
}

func RegisterFunc(prefix string, ginEngine *gin.Engine, clients client.Clients) {
	ginEngine.GET(fmt.Sprintf("%s/healthz", prefix), func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World")
	})
}
