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
// +kubebuilder:validation:rule=`size(filter(self.clusterNetworkConfig.routes, x, toLower(x) == "default")) +
// size(filter(flatten(map(self.podNetworkConfigs, p, p.routes)), x, toLower(x) == "default")) <= 1`
// +kubebuilder:validation:message="Only one default route is allowed across cluster and pod network configurations."
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
// +kubebuilder:validation:rule='self.policyBasedRouting || self.routes.size() > 0'
// +kubebuilder:validation:message="Routes list shouldn't be empty if policybasedRouting is disabled."
type PodNetworkConfig struct {
	// PodNetwork is the name of a PodNetwork resource
	PodNetwork string `json:"podNetwork"`
	// PodIPReservationSize is the number of IP address to statically reserve
	// +kubebuilder:default=0
	PodIPReservationSize int `json:"podIPReservationSize,omitempty"`
	// Routes is a list of routes to add to the Pod through interface assigned to this PodNetwork
	// +kubebuilder:validation:rule=`!(self.routes.all(x, toLower(x) == "delegatedsubnet"))`
	// +kubebuilder:validation:message="DelegatedSubnet is not allowed in cluster network configuration routes."
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
}

// ClusterNetworkConfig describes a template for how to attach the infra network to a Pos
// +kubebuilder:validation:rule='self.policyBasedRouting || self.routes.size() > 0'
// +kubebuilder:validation:message="Routes list shouldn't be empty if policybasedRouting is disabled."
type ClusterNetworkConfig struct {
	// Routes is a list of routes to add to the Pod through interface assigned to the infra network
	// +kubebuilder:validation:rule=`!(self.routes.all(x, toLower(x) == "delegatedsubnet"))`
	// +kubebuilder:validation:message="DelegatedSubnet is not allowed in cluster network configuration routes."
	Routes []string `json:"routes,omitempty"`
	// PolicyBasedRouting is a flag to enable policy based routing
	// +kubebuilder:default=true
	PolicyBasedRouting bool `json:"policyBasedRouting,omitempty"`
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
	Status                   PNIStatus                `json:"status,omitempty"`
	PodNetworkConfigStatuses []PodNetworkConfigStatus `json:"PodNetworkConfigStatuses,omitempty"`
}

// PodNetworkStatus describes the status of a PodNetwork
type PodNetworkConfigStatus struct {
	// +kubebuilder:validation:Optional
	Status         PNIStatus `json:"status,omitempty"`
	PodIPAddresses []string  `json:"podIPAddresses,omitempty"`
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
