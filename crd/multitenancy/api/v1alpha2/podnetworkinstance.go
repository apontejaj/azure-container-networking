//go:build !ignore_uncovered
// +build !ignore_uncovered

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// +kubebuilder:object:root=true

// PodNetworkInstance is the Schema for the PodNetworkInstances API
// +kubebuilder:resource:shortName=pni,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels=managed=
// +kubebuilder:metadata:labels=owner=
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
type PodNetworkInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodNetworkInstanceSpec   `json:"spec,omitempty"`
	Status PodNetworkInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodNetworkInstanceList contains a list of PodNetworkInstance
type PodNetworkInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodNetworkInstance `json:"items"`
}

// PodNetworkConfig describes a template for how to attach a PodNetwork to a Pod
type PodNetworkConfig struct {
	// PodNetwork is the name of a PodNetwork resource
	PodNetwork string `json:"podNetwork"`
	// PodIPReservationSize is the number of IP address to statically reserve
	// +kubebuilder:default=0
	PodIPReservationSize int `json:"podIPReservationSize,omitempty"`
	// Routes is a list of routes to add to the Pod through interface assigned to this PodNetwork
	Routes []string `json:"routes,omitempty"`
}

// ClusterNetworkConfig describes a template for how to attach the infra network to a Pos
type ClusterNetworkConfig struct {
	// Routes is a list of routes to add to the Pod through interface assigned to the infra network
	Routes []string `json:"routes,omitempty"`
}

// PodNetworkInstanceSpec defines the desired state of PodNetworkInstance
type PodNetworkInstanceSpec struct {
	// ClusterNetworkConfig describes how to attach the infra network to a Pod
	// +kubebuilder:validation:Required
	ClusterNetworkConfig ClusterNetworkConfig `json:"clusterNetworkConfig"`
	// PodNetworkConfigs describes each PodNetwork to attach to a single Pod
	// +kubebuilder:validation:Required
	PodNetworkConfigs []PodNetworkConfig `json:"podNetworkConfigs"`
}

// PodNetworkInstanceStatus defines the observed state of PodNetworkInstance
type PodNetworkInstanceStatus struct {
	// +kubebuilder:validation:Optional
	PodIPAddresses     []string             `json:"podIPAddresses,omitempty"`
	Status             PNIStatus            `json:"status,omitempty"`
	PodNetworkStatuses map[string]PNIStatus `json:"podNetworkStatuses,omitempty"`
}

// PNIStatus indicates the status of PNI
// +kubebuilder:validation:Enum=Ready;CreateReservationSetError;PodNetworkNotReady;InsufficientIPAddressesOnSubnet
type PNIStatus string

const (
	PNIStatusReady                           PNIStatus = "Ready"
	PNIStatusCreateReservationSetError       PNIStatus = "CreateReservationSetError"
	PNIStatusPodNetworkNotReady              PNIStatus = "PodNetworkNotReady"
	PNIStatusInsufficientIPAddressesOnSubnet PNIStatus = "InsufficientIPAddressesOnSubnet"
)

// func init() {
// 	SchemeBuilder.Register(&PodNetworkInstance{}, &PodNetworkInstanceList{})
// }
