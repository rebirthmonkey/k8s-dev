package restaurant

import (
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ResourceMetadatas = make(map[string]ResourceMetadata)
)

// +kubebuilder:object:generate=false

type ResourceMetadata struct {
	Type            runtime.Object `json:"-"`
	Desc            string         `json:"desc,omitempty"`
	Catalog         string         `json:"catalog,omitempty"`
	Prefix          string         `json:"-"`
	PhaseMachine    bool           `json:"phaseMachine"`
	PhaseMachineDef interface{}    `json:"-"`
	NoPortable      bool           `json:"-"`
}

func GetResourceMetadata(resource string) *ResourceMetadata {
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

func GetResourceMetadataByResourceType(obj client.Object) (resource string, metadata *ResourceMetadata) {
	typ := reflect.Indirect(reflect.ValueOf(obj)).Type()
	for res, resmeta := range ResourceMetadatas {
		restype := reflect.Indirect(reflect.ValueOf(resmeta.Type)).Type()
		if restype.Name() == typ.Name() {
			return res, &resmeta
		}
	}
	return "", nil
}

// SetPhaseMachine setting PhaseMachine function definition for metadata
func SetPhaseMachine(apiName string, pm interface{}) {
	metadata := ResourceMetadatas[apiName]
	metadata.PhaseMachine = true
	metadata.PhaseMachineDef = pm
	ResourceMetadatas[apiName] = metadata
}
