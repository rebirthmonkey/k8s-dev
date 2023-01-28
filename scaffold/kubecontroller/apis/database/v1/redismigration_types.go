package v1

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/pm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rebirthmonkey/k8s-dev/pkg/apis"
	controllerapis "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/database"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=redismigrations,scope=Namespaced,shortName=rm
// +kubebuilder:subresource:status

// RedisMigration is the Schema for the RedisMigration API.
type RedisMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RedisMigrationSpec   `json:"spec"`
	Status            RedisMigrationStatus `json:"status,omitempty"`
}

// RedisMigrationSpec defines the desired state of RedisMigration
type RedisMigrationSpec struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

// RedisMigrationStatus defines the observed state of RedisMigration
type RedisMigrationStatus struct {
	Phase        pm.Phase                         `json:"phase,omitempty"`
	State        map[pm.Phase]RedisMigrationState `json:"state,omitempty"`
	SourceStatus map[string]string                `json:"sourceStatus,omitempty"`
	DestStatus   map[string]string                `json:"destStatus,omitempty"`
}

type RedisMigrationState struct {
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// +kubebuilder:object:root=true

// RedisMigrationList contains a list of RedisMigration
type RedisMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisMigration `json:"items"`
}

// +teleport:kubebuilder:resource=redismigrations
func init() {
	controllerapis.Register(GroupVersion, &RedisMigration{}, &RedisMigrationList{}).Prioritized()
	controllerapis.Register(database.GroupVersion, &RedisMigration{}, &RedisMigrationList{})
	meta := apis.ResourceMetadata{
		Type:           &RedisMigration{},
		Prefix:         "rm",
		Desc:           "Redis实例迁移",
		GroupVersion:   GroupVersion,
		ControllerType: apis.ControllerTypePhaseMachine,
	}
	apis.ResourceMetadatas[controllerapis.ResourceRedisMigrations] = meta
	apis.ResourcesConfigs.Add(GroupVersion.Group, GroupVersion.Version, controllerapis.ResourceRedisMigrations, apis.ResourceConfig{
		ShortNames:  []string{meta.Prefix},
		NewFunc:     func() runtime.Object { return &RedisMigration{} },
		NewListFunc: func() runtime.Object { return &RedisMigrationList{} },
		//SubResourcesConfig: map[string]reconcilermgr.SubResourceConfig{
		//	apis.SubResourceStatus: {},
		//},
	})
}
