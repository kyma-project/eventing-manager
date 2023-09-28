package eventing

import (
	"fmt"
	"reflect"
	"time"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var natsGVK = schema.GroupVersionResource{
	Group:    natsv1alpha1.GroupVersion.Group,
	Version:  natsv1alpha1.GroupVersion.Version,
	Resource: "nats",
}

func NewWatcher(client dynamic.Interface, namespace string) *NatsWatcher {
	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client, 30*time.Second, namespace, nil)
	dynamicInformer := dynamicInformerFactory.ForResource(natsGVK).Informer()
	return &NatsWatcher{
		client:                 client,
		namespace:              namespace,
		natsCREventsCh:         make(chan event.GenericEvent),
		dynamicInformerFactory: dynamicInformerFactory,
		dynamicInformer:        dynamicInformer,
	}
}

type NatsWatcher struct {
	client                 dynamic.Interface
	namespace              string
	natsCREventsCh         chan event.GenericEvent
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	dynamicInformer        cache.SharedIndexInformer
	eventHandler           cache.ResourceEventHandlerRegistration
	stopCh                 chan struct{}
	started                bool
}

func (w *NatsWatcher) Start() {
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
	if !cache.WaitForNamedCacheSync("NatsWatcher", w.stopCh, w.dynamicInformer.HasSynced) {
		runtime.HandleError(err)
	}

	w.started = true
}

func (w *NatsWatcher) Stop() {
	// recover from closing already closed channels
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic: %v\n", r)
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

func (w *NatsWatcher) addFunc(o interface{}) {
	w.natsCREventsCh <- event.GenericEvent{}
}

func (w *NatsWatcher) updateFunc(o, n interface{}) {
	if !reflect.DeepEqual(o, n) {
		w.natsCREventsCh <- event.GenericEvent{}
	}
}

func (w *NatsWatcher) deleteFunc(o interface{}) {
	w.natsCREventsCh <- event.GenericEvent{}
}
