package apis

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ResourceMetadatas = make(map[string]ResourceMetadata)
)

const (
	// ControllerTypeNone no main controller for the API resource
	ControllerTypeNone = ""
	// ControllerTypeGeneric generic controller implementation
	ControllerTypeGeneric = "generic"
	// ControllerTypePhaseMachine controller implementation based on phasemachine framework
	ControllerTypePhaseMachine = "phasemachine"
)

type ArgumentsModificationRequestable interface {
	RequestModifyArguments()
}

type Resetable interface {
	Reset(ctx context.Context, cli client.Client) error
}

// +kubebuilder:object:generate=false
type ResourceMetadata struct {
	Type            runtime.Object      `json:"-"`
	Desc            string              `json:"desc,omitempty"`
	Catalog         string              `json:"catalog,omitempty"`
	Prefix          string              `json:"-"`
	ControllerType  string              `json:"controllerType"`
	PhaseMachineDef interface{}         `json:"-"`
	GroupVersion    schema.GroupVersion `json:"-"`
	// whether the api resource should only be enabled in tencent intranet deployments
	NoPortable bool `json:"-"`
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
	metadata.ControllerType = ControllerTypePhaseMachine
	metadata.PhaseMachineDef = pm
	ResourceMetadatas[apiName] = metadata
}
