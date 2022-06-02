package main

import (
	"fmt"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	//create config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	//create client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	//get informer
	factory := informers.NewSharedInformerFactory(clientset, 0)
	//factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace("default"))
	informer := factory.Core().V1().Pods().Informer()

	// add workqueue
	rateLimitingQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "controller")

	//add event handler
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				fmt.Println("can not get the key")
			}
			rateLimitingQueue.AddRateLimited(key)
			fmt.Println("Add Event with key：", key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				fmt.Println("can not get the key")
			}
			rateLimitingQueue.AddRateLimited(key)
			fmt.Println("Update Event with key:", key)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				fmt.Println("can not get the key")
			}
			rateLimitingQueue.AddRateLimited(key)
			fmt.Println("Delete Event with key", key)
		},
	})

	//start informer
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	<-stopCh
}
