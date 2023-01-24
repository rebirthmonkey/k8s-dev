package ns

import (
	"context"
	"fmt"
	
	"github.com/gin-gonic/gin"
	"github.com/rebirthmonkey/go/pkg/gin/util"
	"github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr/registry"
	"github.com/rebirthmonkey/k8s-dev/pkg/k8s/client"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	registry.Register(registry.APIExtHandler{
		RegisterFunc: RegisterFunc,
		Prefix:       "/global",
	})
}

func RegisterFunc(prefix string, ginEngine *gin.Engine, clients client.Clients) {
	ginEngine.GET(fmt.Sprintf("%s/nss", prefix), getNSsHandler(clients))
}

func getNSsHandler(clients client.Clients) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespaces := corev1.NamespaceList{}
		cli := clients.KubeClient()
		err := cli.List(context.Background(), &namespaces)
		if err != nil {
			util.WriteResponse(c, err, nil)
		}
		util.WriteResponse(c, nil, namespaces.Items)
	}
}
