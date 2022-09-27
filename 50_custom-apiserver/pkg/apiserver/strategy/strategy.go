package strategy

import (
	"context"
	"github.com/thoas/go-funk"

	"50_custom-apiserver/pkg/apiserver/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource/util"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// STRATEGY HOOKS

// PrepareForCreateFunc hooks into Strategy.PrepareForCreate
// You can modify user provided data here
type PrepareForCreateFunc func(obj runtime.Object)

// PrepareForUpdateFunc hooks into Strategy.PrepareForUpdate.
// typical implementation copies part of new to old resource, the old one
// will then be used to complete the persistence operation. however, you may choose to
// manually modify the old resource and just ignore the new one
// unrestricted: indicates if unrestricted modification is allowed, i.e. status and other
// subresources can be updated when PUTing the MAIN resource. this parameter has no effect
// when the hook is combined with PartialUpdateStrategy
// old: represents current status ( in storage ) of the api resource
// new: represents user provided status of the api resource
type PrepareForUpdateFunc func(unrestricted bool, new, old runtime.Object)

// ValidateFunc hooks into Strategy.Validate, it will be invoked AFTER PrepareForCreateFunc
// to verify if the provided object is ok to create
type ValidateFunc func(ctx context.Context, obj runtime.Object, client ctrlrtclient.Client) field.ErrorList

// ValidateUpdateFunc hooks into Strategy.ValidateUpdate, it will be invoked AFTER PrepareForUpdateFunc
// to verify if the provided object is ok to update
// WARNING: old may have been modified in PrepareForUpdateFunc hook, if you want original object in etcd,
// load it by invoking client
type ValidateUpdateFunc func(ctx context.Context, new, old runtime.Object, client ctrlrtclient.Client) field.ErrorList

// CanonicalizeFunc hooks into Strategy.Canonicalize, it will be invoked AFTER ValidateFunc / ValidateUpdateFunc
// to normalize the provided object
type CanonicalizeFunc func(obj runtime.Object)

// nonNamespacedResources contains all non-namespaced resources
var nonNamespacedResources []string

// AddToNonNameSpaced add resource to nonNamespacedResources slice
func AddToNonNameSpaced(resource string) {
	if nonNamespacedResources == nil {
		nonNamespacedResources = []string{resource}
	} else {
		nonNamespacedResources = append(nonNamespacedResources, resource)
	}
}

// Strategy aggregation of rest create/update/delete strategies
type Strategy interface {
	rest.RESTCreateStrategy
	rest.RESTUpdateStrategy
	rest.RESTDeleteStrategy
}

// DefaultStrategy default adapter for Strategy interface
type DefaultStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
	PrepareForCreateFunc PrepareForCreateFunc
	PrepareForUpdateFunc PrepareForUpdateFunc
	ValidateFunc         ValidateFunc
	ValidateUpdateFunc   ValidateUpdateFunc
	CanonicalizeFunc     CanonicalizeFunc
	ClientFactory        client.KubeClientFactory
	Unrestricted         bool
	Resource             string
}

// WarningsOnCreate get warnings on create
func (s DefaultStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate get warnings on update
func (s DefaultStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// NamespaceScoped tell if the object must be in a namespace or not.
func (s DefaultStrategy) NamespaceScoped() bool {
	return !funk.Contains(nonNamespacedResources, s.Resource)
}

// PrepareForCreate is invoked on create before validation to normalize the object.
func (s DefaultStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if s.PrepareForCreateFunc == nil {
		return
	}
	s.PrepareForCreateFunc(obj)
}

// PrepareForUpdate is invoked on update before validation to normalize the object.
func (s DefaultStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if s.PrepareForUpdateFunc == nil {
		return
	}
	s.PrepareForUpdateFunc(s.Unrestricted, obj, old)
	if err := util.DeepCopy(old, obj); err != nil {
		utilruntime.HandleError(err)
	}
}

// Validate returns an ErrorList with validation errors or nil.
func (s DefaultStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if s.ValidateFunc == nil {
		return field.ErrorList{}
	} else {
		return s.ValidateFunc(ctx, obj, s.ClientFactory())
	}
}

// AllowCreateOnUpdate returns true if the object can be created by a PUT.
func (s DefaultStrategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate returns true if the object can be updated unconditionally
// (irrespective of the latest resource version)
func (s DefaultStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize allows an object to be mutated into a canonical form.
func (s DefaultStrategy) Canonicalize(obj runtime.Object) {
	if s.CanonicalizeFunc != nil {
		s.CanonicalizeFunc(obj)
	}
}

// ValidateUpdate is invoked after default fields in the object have been filled in before the object is persisted.
func (s DefaultStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	if s.ValidateUpdateFunc == nil {
		return field.ErrorList{}
	} else {
		return s.ValidateUpdateFunc(ctx, obj, old, s.ClientFactory())
	}
}

var _ rest.RESTUpdateStrategy = &PartialUpdateStrategy{}

// PartialUpdateStrategy wraps an update strategy, overrides PrepareForUpdate method to invoke
// the provided PrepareForUpdateFunc before update
type PartialUpdateStrategy struct {
	rest.RESTUpdateStrategy
	PrepareForUpdateFunc PrepareForUpdateFunc
	ValidateUpdateFunc   ValidateUpdateFunc
	CanonicalizeFunc     CanonicalizeFunc
	ClientFactory        client.KubeClientFactory
}

// PrepareForUpdate calls the PrepareForUpdate function on obj if supported, otherwise does nothing.
func (s *PartialUpdateStrategy) PrepareForUpdate(ctx context.Context, new, old runtime.Object) {
	if s.PrepareForUpdateFunc != nil {
		s.PrepareForUpdateFunc(false, new, old)
		if err := util.DeepCopy(old, new); err != nil {
			utilruntime.HandleError(err)
		}
	}
}

// ValidateUpdate is invoked after default fields in the object have been
// filled in before the object is persisted.
func (s *PartialUpdateStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	if s.ValidateUpdateFunc == nil {
		return field.ErrorList{}
	} else {
		return s.ValidateUpdateFunc(ctx, obj, old, s.ClientFactory())
	}
}

// Canonicalize allows an object to be mutated into a canonical form.
func (s *PartialUpdateStrategy) Canonicalize(obj runtime.Object) {
	if s.CanonicalizeFunc != nil {
		s.CanonicalizeFunc(obj)
	}
}
