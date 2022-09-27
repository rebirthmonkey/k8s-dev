package informers

import (
	"50_custom-apiserver/pkg/generated/clientset/versioned"
	"50_custom-apiserver/pkg/generated/informers/externalversions"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

const (
	EventTypeAdd    = "Add"
	EventTypeUpdate = "Update"
	EventTypeDelete = "Delete"
)

type EventType string

type Event struct {
	Type   EventType
	Object metav1.Object
}

type Interface interface {
	Start(ch <-chan struct{}) error
	Stop()
	Watch(gvr schema.GroupVersionResource, queue workqueue.Interface)
}

type impl struct {
	informerFactory externalversions.SharedInformerFactory
	watches         map[schema.GroupVersionResource][]workqueue.Interface
}

func New(ip string, port int, token string, defaultResync time.Duration) (Interface, error) {
	insecureUrl := fmt.Sprintf("http://%s:%d", ip, port)
	config, err := clientcmd.BuildConfigFromFlags(insecureUrl, "")
	config.Insecure = true
	config.BearerToken = token
	tcmclient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	informerFactory := externalversions.NewSharedInformerFactory(tcmclient, defaultResync)
	return &impl{
		informerFactory: informerFactory,
		watches:         map[schema.GroupVersionResource][]workqueue.Interface{},
	}, nil
}

func (i *impl) Watch(gvr schema.GroupVersionResource, queue workqueue.Interface) {
	queues, ok := i.watches[gvr]
	if !ok {
		klog.Infof("Informer for %s/%s %s requested", gvr.Group, gvr.Version, gvr.Resource)
	}
	queues = append(queues, queue)
	i.watches[gvr] = queues
}

func (i *impl) Start(stopCh <-chan struct{}) error {
	informerFactory := i.informerFactory

	for gvr := range i.watches {
		gvrstr := fmt.Sprintf("%s/%s %s", gvr.Group, gvr.Version, gvr.Resource)
		genericInformer, err := informerFactory.ForResource(gvr)
		if err != nil {
			return err
		}
		queues := i.watches[gvr]
		genericInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o, ok := obj.(metav1.Object)
				if !ok {
					return
				}
				klog.V(1).Infof("%s event arrived for %s %s", EventTypeAdd, gvrstr, o.GetName())
				event := Event{
					Type:   EventTypeAdd,
					Object: o,
				}
				for _, q := range queues {
					q.Add(event)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o, ok := newObj.(metav1.Object)
				if !ok {
					return
				}
				klog.V(1).Infof("%s event arrived for %s %s", EventTypeUpdate, gvrstr, o.GetName())
				event := Event{
					Type:   EventTypeUpdate,
					Object: o,
				}
				for _, q := range queues {
					q.Add(event)
				}
			},
			DeleteFunc: func(obj interface{}) {
				var o metav1.Object
				switch obj := obj.(type) {
				case cache.DeletedFinalStateUnknown:
					var ok bool
					o, ok = obj.Obj.(metav1.Object)
					if !ok {
						return
					}
				case metav1.Object:
					o = obj
				default:
					return
				}
				klog.V(1).Infof("%s event arrived for %s %s", EventTypeDelete, gvrstr, o.GetName())
				event := Event{
					Type:   EventTypeDelete,
					Object: o,
				}
				for _, q := range queues {
					q.Add(event)
				}
			},
		})
	}

	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)
	return nil
}

func (i *impl) Stop() {
	for _, queues := range i.watches {
		for _, queue := range queues {
			queue.ShutDown()
		}
	}
}
