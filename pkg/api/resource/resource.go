package resource

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/api/strategy"
	"github.com/rebirthmonkey/k8s-dev/pkg/api/tabconv"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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

// ResourceConfig config for specific resource
type ResourceConfig struct {
	// callback to create a new resource
	NewFunc func() runtime.Object
	// callback to create a new resource list
	NewListFunc func() runtime.Object
	// hook on PrepareForCreate
	PrepareForCreateFunc strategy.PrepareForCreateFunc
	// hook on Validate
	ValidateFunc strategy.ValidateFunc
	// hook on PrepareForUpdate
	PrepareForUpdateFunc strategy.PrepareForUpdateFunc
	// hook on ValidateUpdate
	ValidateUpdateFunc strategy.ValidateUpdateFunc
	// hook on Canonicalize
	CanonicalizeFunc strategy.CanonicalizeFunc
	// to create a TableConvertor
	TableConvertorBuilder tabconv.TableConvertorBuilder
	// configurations for sub-resources
	SubResourcesConfig map[string]SubResourceConfig
	ShortNames         []string
}

// SubResourceConfig config for a subresource
type SubResourceConfig struct {
	// hook on PrepareForUpdate
	PrepareForUpdateFunc strategy.PrepareForUpdateFunc
	// hook on ValidateUpdate
	ValidateUpdateFunc strategy.ValidateUpdateFunc
	// hook on Canonicalize
	CanonicalizeFunc strategy.CanonicalizeFunc
}

const (
	// ControllerTypeNone no main controller for the API resource
	ControllerTypeNone = ""
	// ControllerTypeGeneric generic controller implementation
	ControllerTypeGeneric = "generic"
)
