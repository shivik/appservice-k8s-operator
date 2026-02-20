package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{clientset: clientset, config: config}, nil
}

func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	_, err := c.clientset.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	return err
}

func (c *Client) UpdateDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	_, err := c.clientset.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (c *Client) DeleteDeployment(ctx context.Context, namespace, name string) error {
	return c.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateService(ctx context.Context, service *corev1.Service) error {
	_, err := c.clientset.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
	return err
}

func (c *Client) ListPods(ctx context.Context, namespace string, labelSelector string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
}

func (c *Client) WatchDeployment(ctx context.Context, namespace, name string) error {
	watcher, err := c.clientset.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		deployment, ok := event.Object.(*appsv1.Deployment)
		if !ok {
			continue
		}
		fmt.Printf("Event: %s, Deployment: %s, Replicas: %d/%d\n",
			event.Type, deployment.Name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
	}
	return nil
}

func (c *Client) GetConfig() *rest.Config {
	return c.config
}

func IsNotFound(err error) bool {
	return errors.IsNotFound(err)
}

func IgnoreNotFound(err error) error {
	if IsNotFound(err) {
		return nil
	}
	return err
}

func GetNamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// Made with Bob
