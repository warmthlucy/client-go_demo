package model

import (
	"context"
	"fmt"
	v14 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Controller struct {
	Client        kubernetes.Interface
	PodLister     v1.PodLister
	ServiceLister v1.ServiceLister
	Queue         workqueue.RateLimitingInterface
}

func (c *Controller) Add(obj interface{}) {
	c.enqueue(obj)
	fmt.Println("add pod.....")
}
func (c *Controller) Update(oldObj, newObj interface{}) {
	c.enqueue(newObj)
	fmt.Println("update pod......")
}
func (c *Controller) Delete(obj interface{}) {
	service := obj.(*v14.Service)
	ownerReference := v13.GetControllerOf(service)
	if ownerReference == nil {
		return
	}
	if ownerReference.Kind != "Pod" {
		return
	}
	c.enqueue(obj)
	fmt.Println("delete service......")
}

func (c *Controller) syncPod(key string) error {
	fmt.Println(key)
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	//判断pod是否存在
	pod, err := c.PodLister.Pods(namespaceKey).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	//新增和删除
	_, ok := pod.GetLabels()["app"]
	service, err := c.ServiceLister.Services(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if ok && errors.IsNotFound(err) {
		//create service
		sv := c.construct(pod)
		_, err := c.Client.CoreV1().Services(namespaceKey).Create(context.TODO(), sv, v13.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			return err
		}

	} else if !ok && service != nil {
		//delete service
		c.Client.CoreV1().Services(namespaceKey).Delete(context.TODO(), name, v13.DeleteOptions{})
	}

	return nil
}

func (c *Controller) worker() {
	item, shutdown := c.Queue.Get()
	key := item.(string)
	if shutdown {
		return
	}
	defer c.Queue.Done(item)
	c.syncPod(key)
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}

	c.Queue.Add(key)
}

func (c *Controller) construct(pod *v14.Pod) *v14.Service {
	service := &v14.Service{
		ObjectMeta: v13.ObjectMeta{
			OwnerReferences: []v13.OwnerReference{
				*v13.NewControllerRef(pod, v14.SchemeGroupVersion.WithKind("pod")),
			},
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Spec: v14.ServiceSpec{
			Ports: []v14.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	return service
}

func (c *Controller) Run(stopCh chan struct{}) {
	go wait.Until(c.worker, time.Minute, stopCh)
	<-stopCh
}
