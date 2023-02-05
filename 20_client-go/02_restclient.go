package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	//config, err := clientcmd.BuildConfigFromFlags("", "/tmp/cls-j7xitnn8-config")
	if err != nil {
		panic(err)
	}

	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		fmt.Println(err)
	}

	pod := corev1.Pod{}
	err = restClient.Get().
		Namespace("default").
		Resource("pods").
		Name("test").
		Do(context.TODO()).
		Into(&pod)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(pod.Status)
	}
}
