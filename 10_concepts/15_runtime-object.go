package main

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func main() {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"name": "foo"},
		},
	}

	podObj := runtime.Object(pod)

	pod2, ok := podObj.(*corev1.Pod)
	if !ok {
		fmt.Println("the conversion is unexpected")
	}

	if !reflect.DeepEqual(pod, pod2) {
		fmt.Println("the conversion is unexpected")
	}
}
