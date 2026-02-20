package controllers

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1alpha1 "github.com/example/k8s-operator/api/v1alpha1"
)

type AppServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	appService := &examplev1alpha1.AppService{}
	if err := r.Get(ctx, req.NamespacedName, appService); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if appService.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(appService, "example.com/finalizer") {
			controllerutil.AddFinalizer(appService, "example.com/finalizer")
			if err := r.Update(ctx, appService); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(appService, "example.com/finalizer") {
			if err := r.cleanupResources(ctx, appService); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(appService, "example.com/finalizer")
			if err := r.Update(ctx, appService); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: appService.Name, Namespace: appService.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForAppService(appService)
		logger.Info("Creating Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if *deployment.Spec.Replicas != appService.Spec.Replicas {
		deployment.Spec.Replicas = &appService.Spec.Replicas
		if err = r.Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: appService.Name, Namespace: appService.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		svc := r.serviceForAppService(appService)
		logger.Info("Creating Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		if err = r.Create(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	appService.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	appService.Status.LastReconcileTime = metav1.Now()

	if deployment.Status.AvailableReplicas == appService.Spec.Replicas {
		appService.Status.Phase = "Running"
		r.updateCondition(appService, "Ready", metav1.ConditionTrue, "DeploymentReady", "All replicas are available")
	} else {
		appService.Status.Phase = "Pending"
		r.updateCondition(appService, "Ready", metav1.ConditionFalse, "DeploymentNotReady", "Waiting for replicas")
	}

	if err := r.Status().Update(ctx, appService); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *AppServiceReconciler) deploymentForAppService(app *examplev1alpha1.AppService) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	replicas := app.Spec.Replicas

	containers := []corev1.Container{{
		Name:  app.Name,
		Image: app.Spec.Image,
		Ports: []corev1.ContainerPort{{ContainerPort: app.Spec.Port}},
	}}

	if len(app.Spec.Environment) > 0 {
		envVars := []corev1.EnvVar{}
		for k, v := range app.Spec.Environment {
			envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
		}
		containers[0].Env = envVars
	}

	if app.Spec.Resources.CPU != "" || app.Spec.Resources.Memory != "" {
		containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{Containers: containers},
			},
		},
	}

	controllerutil.SetControllerReference(app, dep, r.Scheme)
	return dep
}

func (r *AppServiceReconciler) serviceForAppService(app *examplev1alpha1.AppService) *corev1.Service {
	labels := map[string]string{"app": app.Name}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       app.Spec.Port,
				TargetPort: intstr.FromInt(int(app.Spec.Port)),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	controllerutil.SetControllerReference(app, svc, r.Scheme)
	return svc
}

func (r *AppServiceReconciler) updateCondition(app *examplev1alpha1.AppService, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	for i, c := range app.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				app.Status.Conditions[i] = condition
			}
			return
		}
	}
	app.Status.Conditions = append(app.Status.Conditions, condition)
}

func (r *AppServiceReconciler) cleanupResources(ctx context.Context, app *examplev1alpha1.AppService) error {
	logger := log.FromContext(ctx)
	logger.Info("Cleaning up resources", "AppService", app.Name)
	return nil
}

func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.AppService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// Made with Bob
