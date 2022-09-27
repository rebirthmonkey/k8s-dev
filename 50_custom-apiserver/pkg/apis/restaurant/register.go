/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package restaurant

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"50_custom-apiserver/pkg/apiserver"
)

// GroupName is the group name used in this package
const GroupName = "restaurant.wukong.com"

// +kubebuilder:object:generate=false

type ResourcesConfig map[string]map[string]apiserver.ResourceConfig

func (rc ResourcesConfig) Add(version, resource string, config apiserver.ResourceConfig) {
	rcv, ok := rc[version]
	if !ok {
		rcv = make(map[string]apiserver.ResourceConfig)
		rc[version] = rcv
	}
	rcv[resource] = config
}

// SchemeGroupVersion is group version used to register these objects
var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

	//// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	//// no &scheme.Builder{} here, otherwise vk __internal/WatchEvent will double registered to k8s.io/apimachinery/pkg/apis/meta/v1.WatchEvent &
	//// k8s.io/apimachinery/pkg/apis/meta/v1.InternalEvent, which is illegal
	//SchemeBuilder = runtime.NewSchemeBuilder()
	//
	//// AddToScheme adds the types in this group-version to the given scheme.
	//AddToScheme = SchemeBuilder.AddToScheme

	AllResourcesConfig = make(ResourcesConfig)
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder is the scheme builder with scheme init functions to run for this API package
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is a common registration function for mapping packaged scoped group & version keys to a scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// Adds the list of known types to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Pizza{},
		&PizzaList{},
		&Topping{},
		&ToppingList{},
	)
	return nil
}
