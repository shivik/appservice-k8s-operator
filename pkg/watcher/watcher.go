package watcher

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type EventType string

const (
	EventAdded    EventType = "ADDED"
	EventModified EventType = "MODIFIED"
	EventDeleted  EventType = "DELETED"
	EventError    EventType = "ERROR"
)

type Event struct {
	Type   EventType
	Object runtime.Object
}

type EventHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj, newObj interface{})
	OnDelete(obj interface{})
}

type ResourceWatcher struct {
	client   client.Client
	informer cache.SharedIndexInformer
	stopCh   chan struct{}
	handlers []EventHandler
}

func NewResourceWatcher(c client.Client, obj client.Object, resyncPeriod time.Duration) *ResourceWatcher {
	return &ResourceWatcher{
		client:   c,
		stopCh:   make(chan struct{}),
		handlers: []EventHandler{},
	}
}

func (w *ResourceWatcher) AddEventHandler(handler EventHandler) {
	w.handlers = append(w.handlers, handler)
}

func (w *ResourceWatcher) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting resource watcher")

	if w.informer != nil {
		w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				for _, h := range w.handlers {
					h.OnAdd(obj)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				for _, h := range w.handlers {
					h.OnUpdate(oldObj, newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				for _, h := range w.handlers {
					h.OnDelete(obj)
				}
			},
		})

		go w.informer.Run(w.stopCh)

		if !cache.WaitForCacheSync(w.stopCh, w.informer.HasSynced) {
			return fmt.Errorf("failed to sync cache")
		}
	}

	return nil
}

func (w *ResourceWatcher) Stop() {
	close(w.stopCh)
}

type SimpleEventHandler struct {
	OnAddFunc    func(obj interface{})
	OnUpdateFunc func(oldObj, newObj interface{})
	OnDeleteFunc func(obj interface{})
}

func (h *SimpleEventHandler) OnAdd(obj interface{}) {
	if h.OnAddFunc != nil {
		h.OnAddFunc(obj)
	}
}

func (h *SimpleEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if h.OnUpdateFunc != nil {
		h.OnUpdateFunc(oldObj, newObj)
	}
}

func (h *SimpleEventHandler) OnDelete(obj interface{}) {
	if h.OnDeleteFunc != nil {
		h.OnDeleteFunc(obj)
	}
}

type WatchManager struct {
	watchers map[string]*ResourceWatcher
}

func NewWatchManager() *WatchManager {
	return &WatchManager{
		watchers: make(map[string]*ResourceWatcher),
	}
}

func (wm *WatchManager) AddWatcher(name string, watcher *ResourceWatcher) {
	wm.watchers[name] = watcher
}

func (wm *WatchManager) StartAll(ctx context.Context) error {
	for name, watcher := range wm.watchers {
		if err := watcher.Start(ctx); err != nil {
			return fmt.Errorf("failed to start watcher %s: %w", name, err)
		}
	}
	return nil
}

func (wm *WatchManager) StopAll() {
	for _, watcher := range wm.watchers {
		watcher.Stop()
	}
}

func ConvertToEvent(watchEvent watch.Event) Event {
	var eventType EventType
	switch watchEvent.Type {
	case watch.Added:
		eventType = EventAdded
	case watch.Modified:
		eventType = EventModified
	case watch.Deleted:
		eventType = EventDeleted
	case watch.Error:
		eventType = EventError
	default:
		eventType = EventError
	}

	return Event{
		Type:   eventType,
		Object: watchEvent.Object,
	}
}

func GetObjectMeta(obj interface{}) (metav1.Object, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	return accessor, nil
}

func GetObjectName(obj interface{}) string {
	meta, err := GetObjectMeta(obj)
	if err != nil {
		return ""
	}
	return meta.GetName()
}

func GetObjectNamespace(obj interface{}) string {
	meta, err := GetObjectMeta(obj)
	if err != nil {
		return ""
	}
	return meta.GetNamespace()
}

// Made with Bob
