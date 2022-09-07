package main

import (
	"context"
	"fmt"
	"log"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	v1 "wukong.com/pkg/apis/wukong.com/v1"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}

	config.APIPath = "/apis/"
	config.NegotiatedSerializer = v1.Codecs.WithoutConversion()
	config.GroupVersion = &v1.GroupVersion

	client, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalln(err)
	}

	foo := v1.Foo{}
	err = client.Get().Namespace("default").Resource("foos").Name("crd-test").Do(context.TODO()).Into(&foo)
	if err != nil {
		log.Fatalln(err)
	}

	newObj := foo.DeepCopy()
	newObj.Spec.Name = "test2"

	fmt.Println(foo.Spec.Name)
	fmt.Println(foo.Spec.Replicas)

	fmt.Println(newObj.Spec.Name)
}
