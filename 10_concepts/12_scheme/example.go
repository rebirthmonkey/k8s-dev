package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	coreGV := schema.GroupVersion{Group: "", Version: "v1"}
	extensionsGV := schema.GroupVersion{Group: "extensions", Version: "v1beta1"}
	coreInternalGV := schema.GroupVersion{Group: "", Version: runtime.APIVersionInternal}
	Unversioned := schema.GroupVersion{Group: "", Version: "v1"}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreGV, &corev1.Pod{})
	scheme.AddKnownTypes(extensionsGV, &appsv1.DaemonSet{})
	scheme.AddKnownTypes(coreInternalGV, &corev1.Pod{})
	scheme.AddUnversionedTypes(Unversioned, &metav1.Status{})

	fmt.Println("all known types are:", scheme.AllKnownTypes())
}
