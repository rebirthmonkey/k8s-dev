package apis

import "k8s.io/apimachinery/pkg/runtime"

// ResourceConfig config for specific resource
type ResourceConfig struct {
	// callback to create a new resource
	NewFunc func() runtime.Object
	// callback to create a new resource list
	NewListFunc func() runtime.Object
	//// hook on PrepareForCreate
	//PrepareForCreateFunc strategy.PrepareForCreateFunc
	//// hook on Validate
	//ValidateFunc strategy.ValidateFunc
	//// hook on PrepareForUpdate
	//PrepareForUpdateFunc strategy.PrepareForUpdateFunc
	//// hook on ValidateUpdate
	//ValidateUpdateFunc strategy.ValidateUpdateFunc
	//// hook on Canonicalize
	//CanonicalizeFunc strategy.CanonicalizeFunc
	//// to create a TableConvertor
	//TableConvertorBuilder tabconv.TableConvertorBuilder
	//// configurations for sub-resources
	//SubResourcesConfig map[string]SubResourceConfig
	ShortNames []string
}
