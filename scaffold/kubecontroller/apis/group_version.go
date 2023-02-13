package apis

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	groupVersions         []*groupVersion
	schemeBuilder         = runtime.NewSchemeBuilder()
	internalSchemeBuilder = runtime.NewSchemeBuilder()
	AddToScheme           = schemeBuilder.AddToScheme
	AddInternalToScheme   = internalSchemeBuilder.AddToScheme
)

func init() {
	reconcilermgr.AddToScheme = schemeBuilder.AddToScheme
}

type groupVersion struct {
	schema.GroupVersion
	prioritized bool
}

func (gv *groupVersion) Prioritized() {
	gv.prioritized = true
}

func Register(gv schema.GroupVersion, types ...runtime.Object) *groupVersion {
	gv_ := uniqueGroupVersion(gv, gv.Version == runtime.APIVersionInternal)
	sb := &schemeBuilder
	if gv.Version == runtime.APIVersionInternal {
		sb = &internalSchemeBuilder
	}
	(*sb).Register(func(scheme *runtime.Scheme) error {
		if gv.Version != runtime.APIVersionInternal {
			metav1.AddToGroupVersion(scheme, gv_.GroupVersion)
		}
		scheme.AddKnownTypes(gv_.GroupVersion, types...)
		return nil
	})
	return gv_
}

func GroupVersions() []schema.GroupVersion {
	ret := make([]schema.GroupVersion, len(groupVersions))
	for i, item := range groupVersions {
		ret[i] = item.GroupVersion
	}
	return ret
}

func PrioritizedGroupVersions() []schema.GroupVersion {
	var ret []schema.GroupVersion
	for _, item := range groupVersions {
		if item.prioritized && item.Version != runtime.APIVersionInternal {
			ret = append(ret, item.GroupVersion)
		}
	}
	return ret
}

func uniqueGroupVersion(gv schema.GroupVersion, prioritized bool) *groupVersion {
	var current *groupVersion
	for _, item := range groupVersions {
		if !(gv.Group == item.Group && gv.Version == item.Version) {
			continue
		}
		current = item
		break
	}
	if current == nil {
		current = &groupVersion{gv, prioritized}
		groupVersions = append(groupVersions, current)
	}
	return current
}
