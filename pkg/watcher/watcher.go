package watcher

import (
	"log"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NewResourceWatcher(client dynamic.Interface, gvk schema.GroupVersionResource, namespace string) *ResourceWatcher {
	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client, 30*time.Second, namespace, nil)
	dynamicInformer := dynamicInformerFactory.ForResource(gvk).Informer()
	return &ResourceWatcher{
		client:                 client,
		namespace:              namespace,
		eventsCh:               make(chan event.GenericEvent),
		dynamicInformerFactory: dynamicInformerFactory,
		dynamicInformer:        dynamicInformer,
	}
}

//go:generate go run github.com/vektra/mockery/v2 --name=Watcher --outpkg=mocks --case=underscore
type Watcher interface {
	Start()
	Stop()
	GetEventsChannel() <-chan event.GenericEvent
	IsStarted() bool
}

type ResourceWatcher struct {
	client                 dynamic.Interface
	namespace              string
	eventsCh               chan event.GenericEvent
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	dynamicInformer        cache.SharedIndexInformer
	eventHandler           cache.ResourceEventHandlerRegistration
	stopCh                 chan struct{}
	started                bool
}

func (w *ResourceWatcher) Start() {
	if w.started {
		return
	}

	defer runtime.HandleCrash()

	w.stopCh = make(chan struct{})
	var err error
	w.eventHandler, err = w.dynamicInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    w.addFunc,
		UpdateFunc: w.updateFunc,
		DeleteFunc: w.deleteFunc,
	})

	if err != nil {
		runtime.HandleError(err)
	}

	w.dynamicInformerFactory.Start(w.stopCh)
	w.dynamicInformerFactory.WaitForCacheSync(w.stopCh)
	if !cache.WaitForNamedCacheSync("ResourceWatcher", w.stopCh, w.dynamicInformer.HasSynced) {
		runtime.HandleError(err)
	}

	w.started = true
}

func (w *ResourceWatcher) Stop() {
	// recover from closing already closed channels
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic: %v", r)
		}
	}()

	err := w.dynamicInformer.RemoveEventHandler(w.eventHandler)
	if err != nil {
		runtime.HandleError(err)
	}

	w.stopCh <- struct{}{}
	close(w.stopCh)

	w.dynamicInformerFactory.Shutdown()

	w.started = false
}

func (w *ResourceWatcher) IsStarted() bool {
	return w.started
}

func (w *ResourceWatcher) GetEventsChannel() <-chan event.GenericEvent {
	return w.eventsCh
}

func (w *ResourceWatcher) addFunc(o interface{}) {
	w.eventsCh <- event.GenericEvent{}
}

func (w *ResourceWatcher) updateFunc(o, n interface{}) {
	if !reflect.DeepEqual(o, n) {
		w.eventsCh <- event.GenericEvent{}
	}
}

func (w *ResourceWatcher) deleteFunc(o interface{}) {
	w.eventsCh <- event.GenericEvent{}
}
