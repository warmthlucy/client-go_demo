package controller

import (
	"client-go-demo/model"
	_ "fmt"
	_ "k8s.io/apimachinery/pkg/util/runtime"
	informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func NewController(client kubernetes.Interface, podInformer informer.PodInformer, serviceInformer informer.ServiceInformer) *model.Controller {
	c := &model.Controller{
		Client:        client,
		PodLister:     podInformer.Lister(),
		ServiceLister: serviceInformer.Lister(),
		Queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	// add event handler
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.Add,
		UpdateFunc: c.Update,
	})
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.Delete,
	})

	return c
}
