package v1

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/apis"
	"github.com/rebirthmonkey/k8s-dev/pkg/pm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	controllerapis "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=banana,scope=Namespaced,shortName=ba
// +kubebuilder:subresource:status

// Banana is the Schema for the RedisMigration API.
type Banana struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BananaSpec   `json:"spec"`
	Status            BananaStatus `json:"status,omitempty"`
}

// BananaSpec defines the desired state of RedisMigration
type BananaSpec struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

// BananaStatus defines the observed state of RedisMigration
type BananaStatus struct {
	Phase        pm.Phase                 `json:"phase,omitempty"`
	State        map[pm.Phase]BananaState `json:"state,omitempty"`
	SourceStatus map[string]string        `json:"sourceStatus,omitempty"`
	DestStatus   map[string]string        `json:"destStatus,omitempty"`
}

type BananaState struct {
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// +kubebuilder:object:root=true

// BananaList contains a list of Banana
type BananaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Banana `json:"items"`
}

// +teleport:kubebuilder:resource=bananas
func init() {
	controllerapis.Register(GroupVersion, &Banana{}, &BananaList{}).Prioritized()
	controllerapis.Register(demo.GroupVersion, &Banana{}, &BananaList{})
	meta := apis.ResourceMetadata{
		Type:           &Banana{},
		Prefix:         "ba",
		Desc:           "banana description",
		GroupVersion:   GroupVersion,
		ControllerType: apis.ControllerTypePhaseMachine,
	}
	apis.ResourceMetadatas[controllerapis.ResourceBananas] = meta
	apis.ResourcesConfigs.Add(GroupVersion.Group, GroupVersion.Version, controllerapis.ResourceBananas, apis.ResourceConfig{
		ShortNames:  []string{meta.Prefix},
		NewFunc:     func() runtime.Object { return &Banana{} },
		NewListFunc: func() runtime.Object { return &BananaList{} },
	})
}
