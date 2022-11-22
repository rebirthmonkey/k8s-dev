package apis

import (
	"fmt"
	"reflect"

	apiresource "github.com/rebirthmonkey/k8s-dev/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false
type ResourcesConfigsMap map[string]map[string]apiresource.ResourceConfig

var (
	ResourceMetadatas = make(map[string]apiresource.ResourceMetadata)
	ResourcesConfigs  = make(ResourcesConfigsMap)
)

func GetResourceMetadata(resource string) *apiresource.ResourceMetadata {
	resmeta, ok := ResourceMetadatas[resource]
	if ok {
		return &resmeta
	} else {
		for _, resmeta := range ResourceMetadatas {
			if resmeta.Prefix == resource {
				return &resmeta
			}
		}
		return nil
	}
}

func GetResourceMetadataByResourceType(obj client.Object) (resource string, metadata *apiresource.ResourceMetadata) {
	typ := reflect.Indirect(reflect.ValueOf(obj)).Type()
	for res, resmeta := range ResourceMetadatas {
		restype := reflect.Indirect(reflect.ValueOf(resmeta.Type)).Type()
		if restype.Name() == typ.Name() {
			return res, &resmeta
		}
	}
	return "", nil
}

func (rc ResourcesConfigsMap) Add(group, version, resource string, config apiresource.ResourceConfig) {
	gv := fmt.Sprintf("%s/%s", group, version)
	rcv, ok := rc[gv]
	if !ok {
		rcv = make(map[string]apiresource.ResourceConfig)
		rc[gv] = rcv
	}
	rcv[resource] = config
}
