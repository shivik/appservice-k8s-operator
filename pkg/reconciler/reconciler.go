package reconciler

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type BaseReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (r *BaseReconciler) HandleError(ctx context.Context, err error, msg string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if err != nil {
		logger.Error(err, msg)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BaseReconciler) RequeueAfter(duration time.Duration) ctrl.Result {
	return ctrl.Result{RequeueAfter: duration}
}

func (r *BaseReconciler) Requeue() ctrl.Result {
	return ctrl.Result{Requeue: true}
}

func (r *BaseReconciler) Done() ctrl.Result {
	return ctrl.Result{}
}

func (r *BaseReconciler) GetObject(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	if err := r.Client.Get(ctx, key, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *BaseReconciler) CreateOrUpdate(ctx context.Context, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	existing := obj.DeepCopyObject().(client.Object)

	err := r.Client.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Client.Create(ctx, obj)
		}
		return err
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	return r.Client.Update(ctx, obj)
}

func (r *BaseReconciler) DeleteIfExists(ctx context.Context, obj client.Object) error {
	err := r.Client.Delete(ctx, obj)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

type ReconcileResult struct {
	Requeue      bool
	RequeueAfter time.Duration
	Error        error
}

func (rr ReconcileResult) ToCtrlResult() (ctrl.Result, error) {
	return ctrl.Result{
		Requeue:      rr.Requeue,
		RequeueAfter: rr.RequeueAfter,
	}, rr.Error
}

func Success() ReconcileResult {
	return ReconcileResult{}
}

func SuccessWithRequeue(after time.Duration) ReconcileResult {
	return ReconcileResult{RequeueAfter: after}
}

func Error(err error) ReconcileResult {
	return ReconcileResult{Error: err}
}

func ErrorWithRequeue(err error) ReconcileResult {
	return ReconcileResult{Requeue: true, Error: err}
}

type ReconcilePhase string

const (
	PhaseInitializing ReconcilePhase = "Initializing"
	PhaseReconciling  ReconcilePhase = "Reconciling"
	PhaseReady        ReconcilePhase = "Ready"
	PhaseFailed       ReconcilePhase = "Failed"
	PhaseDeleting     ReconcilePhase = "Deleting"
)

func LogReconcileStart(ctx context.Context, name, namespace string) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "name", name, "namespace", namespace)
}

func LogReconcileEnd(ctx context.Context, name, namespace string, result ctrl.Result, err error) {
	logger := log.FromContext(ctx)
	if err != nil {
		logger.Error(err, "Reconciliation failed", "name", name, "namespace", namespace)
	} else {
		logger.Info("Reconciliation completed", "name", name, "namespace", namespace, "requeue", result.Requeue)
	}
}

func WrapReconcile(ctx context.Context, name, namespace string, fn func() (ctrl.Result, error)) (ctrl.Result, error) {
	LogReconcileStart(ctx, name, namespace)
	result, err := fn()
	LogReconcileEnd(ctx, name, namespace, result, err)
	return result, err
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
	Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{})
}

func RecordEvent(recorder EventRecorder, obj runtime.Object, eventType, reason, message string) {
	if recorder != nil {
		recorder.Event(obj, eventType, reason, message)
	}
}

func RecordEventf(recorder EventRecorder, obj runtime.Object, eventType, reason, messageFmt string, args ...interface{}) {
	if recorder != nil {
		recorder.Eventf(obj, eventType, reason, messageFmt, args...)
	}
}

func FormatError(operation string, err error) error {
	return fmt.Errorf("failed to %s: %w", operation, err)
}

// Made with Bob
