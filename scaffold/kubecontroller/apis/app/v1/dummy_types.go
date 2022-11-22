package v1

import (
	apiresource "github.com/rebirthmonkey/k8s-dev/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/app"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=dummies,scope=Namespaced,shortName=dummy
// +kubebuilder:subresource:status

// Dummy is the Schema for the Dummy API.
type Dummy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DummySpec   `json:"spec"`
	Status            DummyStatus `json:"status,omitempty"`
}

// DummySpec defines the desired state of Dummy
type DummySpec struct {
	TransitionDefer int    `json:"transitionDefer,omitempty"`
	Data            string `json:"data,omitempty"`
}

// DummyStatus defines the observed state of Dummy
type DummyStatus struct {
	Data string `json:"data,omitempty"`
}

// +kubebuilder:object:root=true

// DummyList contains a list of Dummy
type DummyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dummy `json:"items"`
}

// +teleport:kubebuilder:resource=dummies
func init() {
	apis.Register(GroupVersion, &Dummy{}, &DummyList{}).Prioritized()
	apis.Register(app.GroupVersion, &Dummy{}, &DummyList{})
	meta := apiresource.ResourceMetadata{
		Type:           &Dummy{},
		Desc:           "测试",
		Catalog:        "others",
		Prefix:         "dummy",
		ControllerType: apiresource.ControllerTypeGeneric,
		GroupVersion:   GroupVersion,
	}
	apis.ResourceMetadatas[apis.ResourceDummys] = meta
	apis.ResourcesConfigs.Add(GroupVersion.Group, GroupVersion.Version, apis.ResourceDummys, apiresource.ResourceConfig{
		ShortNames:  []string{meta.Prefix},
		NewFunc:     func() runtime.Object { return &Dummy{} },
		NewListFunc: func() runtime.Object { return &DummyList{} },
		SubResourcesConfig: map[string]apiresource.SubResourceConfig{
			apis.SubResourceStatus: {},
		},
	})
}
