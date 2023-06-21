package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Arman struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArmanSpec   `json:"spec,omitempty"`
	Status ArmanStatus `json:"status,omitempty"`
}

type ArmanStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

type ArmanSpec struct {
	DeploymentName    string `json:"deploymentName"`
	DeploymentImage   string `json:"deploymentImage"`
	Replicas          *int32 `json:"replicas"`
	ServiceName       string `json:"serviceName"`
	ServicePort       int32  `json:"servicePort"`
	ServiceType       string `json:"serviceType"`
	ServiceTargetPort int32  `json:"serviceTargetPort"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ArmanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Arman `json:"items,omitempty"`
}
