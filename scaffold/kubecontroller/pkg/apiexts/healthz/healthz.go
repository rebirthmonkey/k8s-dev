package healthz

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr/registry"
	"net/http"
)

func init() {
	registry.Register(registry.APIExtHandler{
		RegisterFunc: RegisterFunc,
		Prefix:       "/v1",
	})
}

func RegisterFunc(prefix string, ginEngine *gin.Engine) {
	ginEngine.GET(fmt.Sprintf("%s/healthz", prefix), func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World")
	})
}
