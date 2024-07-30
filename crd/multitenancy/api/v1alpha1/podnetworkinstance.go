//go:build !ignore_uncovered
// +build !ignore_uncovered

package v1alpha1

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
// +kubebuilder:validation:XValidation:rule="size(filter(self.clusterNetworkConfig.routes, x, toLower(x) == 'default')) + size(filter(flatten(map(self.podNetworkConfigs, p, p.routes)), x, toLower(x) == 'default')) <= 1",message="Only one default route is allowed across cluster and pod network configurations."
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
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
}

// ClusterNetworkConfig describes a template for how to attach the infra network to a Pos
// +kubebuilder:validation:XValidation:rule="self.policyBasedRouting || self.routes.size() > 0",message="Routes list shouldn't be empty if policybasedRouting is disabled."
type ClusterNetworkConfig struct {
	// Routes is a list of routes to add to the Pod through interface assigned to the infra network
	// +kubebuilder:validation:XValidation:rule=`!(self.routes.all(x, toLower(x) == "delegatedsubnet"))`,message="DelegatedSubnet is not allowed in cluster network configuration routes."
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
}

// PodNetworkInstanceSpec defines the desired state of PodNetworkInstance
type PodNetworkInstanceSpec struct {
	// Deprecated - use PodNetworks
	// +kubebuilder:validation:Optional
	PodNetwork string `json:"podnetwork,omitempty"`
	// Deprecated - use PodNetworks
	// +kubebuilder:default=0
	PodIPReservationSize int `json:"podIPReservationSize,omitempty"`
	// PodNetworkConfigs describes each PodNetwork to attach to a single Pod
	// optional for now in case orchestrator uses the deprecated fields
	// +kubebuilder:validation:Optional
	PodNetworkConfigs []PodNetworkConfig `json:"podNetworkConfigs"`
	// ClusterNetworkConfig describes how to attach the infra network to a Pod.
	// optional for now in case orchestrator uses the deprecated fields
	// +kubebuilder:validation:Optional
	ClusterNetworkConfig ClusterNetworkConfig `json:"clusterNetworkConfig"`
}

// PodNetworkStatus describes the status of a PodNetwork
type NetworkStatus struct {
	Network        string    `json:"network,omitempty"`
	Status         PNIStatus `json:"status,omitempty"`
	PodIPAddresses []string  `json:"podIPAddresses,omitempty"`
}

// PodNetworkInstanceStatus defines the observed state of PodNetworkInstance
type PodNetworkInstanceStatus struct {
	// +kubebuilder:validation:Optional
	// Deprecated - use PodNetworkConfigStatuses
	PodIPAddresses []string  `json:"podIPAddresses,omitempty"`
	Status         PNIStatus `json:"status,omitempty"`
	// Deprecated - use PodNetworkConfigStatuses
	PodNetworkStatuses map[string]PNIStatus `json:"podNetworkStatuses,omitempty"`
	// +kubebuilder:validation:Optional
	NetworkStatuses []NetworkStatus `json:"networkStatuses,omitempty"`
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
