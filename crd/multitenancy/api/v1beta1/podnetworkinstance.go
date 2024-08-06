//go:build !ignore_uncovered
// +build !ignore_uncovered

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// +kubebuilder:object:root=true

// PodNetworkInstance is the Schema for the PodNetworkInstances API
// +kubebuilder:resource:shortName=pni,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels=managed=
// +kubebuilder:metadata:labels=owner=
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:unservedversion
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
// +kubebuilder:validation:XValidation:rule="self.policyBasedRouting || self.routes.size() > 0",message="Routes list shouldn't be empty if policybasedRouting is disabled."
type PodNetworkConfig struct {
	// PodNetwork is the name of a PodNetwork resource
	PodNetwork string `json:"podNetwork"`
	// PodIPReservationSize is the number of IP address to statically reserve
	// +kubebuilder:default=0
	PodIPReservationSize int `json:"podIPReservationSize,omitempty"`
	// Routes is a list of routes to add to the Pod through interface assigned to this PodNetwork
	// +kubebuilder:default={}
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
}

// ClusterNetworkConfig describes a template for how to attach the infra network to a Pod
// +kubebuilder:validation:XValidation:rule="self.policyBasedRouting || self.routes.size() > 0",message="Routes list shouldn't be empty if policybasedRouting is disabled."
type ClusterNetworkConfig struct {
	// +kubebuilder:default={}
	// Routes is a list of routes to add to the Pod through interface assigned to the infra network
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
}

// PodNetworkInstanceSpec defines the desired state of PodNetworkInstance
type PodNetworkInstanceSpec struct {
	// ClusterNetworkConfig describes how to attach the infra network to a Pod
	ClusterNetworkConfig ClusterNetworkConfig `json:"clusterNetworkConfig"`
	// PodNetworkConfigs describes each PodNetwork to attach to a single Pod
	PodNetworkConfigs []PodNetworkConfig `json:"podNetworkConfigs"`
}

// PodNetworkConfigStatus is the status of the PodNetworkConfig
type PodNetworkConfigStatus struct {
	// +kubebuilder:validation:Optional
	Status PNIStatus `json:"status,omitempty"`
	// +kubebuilder:validation:Optional
	PodIPAddresses []string `json:"podIPAddresses,omitempty"`
}

// PodNetworkInstanceStatus defines the observed state of PodNetworkInstance
type PodNetworkInstanceStatus struct {
	// Status indicates the status of PNI
	Status PNIStatus `json:"status,omitempty"`
	// PodNetworkConfigStatuses describes the status of each PodNetworkConfig
	// +kubebuilder:validation:Optional
	PodNetworkConfigStatuses []PodNetworkConfigStatus `json:"podNetworkConfigStatuses,omitempty"`
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

func init() {
	SchemeBuilder.Register(&PodNetworkInstance{}, &PodNetworkInstanceList{})
}
