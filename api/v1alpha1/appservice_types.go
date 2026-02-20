package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppServiceSpec struct {
	Replicas    int32                `json:"replicas"`
	Image       string               `json:"image"`
	Port        int32                `json:"port"`
	Environment map[string]string    `json:"environment,omitempty"`
	Resources   ResourceRequirements `json:"resources,omitempty"`
}

type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type AppServiceStatus struct {
	Phase             string             `json:"phase"`
	AvailableReplicas int32              `json:"availableReplicas"`
	Conditions        []metav1.Condition `json:"conditions,omitempty"`
	LastReconcileTime metav1.Time        `json:"lastReconcileTime,omitempty"`
}

type AppService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AppServiceSpec   `json:"spec"`
	Status            AppServiceStatus `json:"status,omitempty"`
}

type AppServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppService{}, &AppServiceList{})
}

// Made with Bob
