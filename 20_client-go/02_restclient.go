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

	// config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		println(err)
	}

	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs

	// REST client
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		println(err)
	}

	// get data
	pod := corev1.Pod{}
	err = restClient.Get().
		Namespace("default").
		Resource("pods").
		Name("test").
		Do(context.TODO()).
		Into(&pod)
	if err != nil {
		panic(err)
	} else {
		fmt.Println(pod.Status)
	}
}
