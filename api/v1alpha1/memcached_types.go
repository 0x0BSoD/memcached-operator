package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerboseLevel
// +kubebuilder:validation:Enum=Disable;Enable;Moar;Extreme
type VerboseLevel string

const (
	// Disable no verbose output at all
	Disable VerboseLevel = "Disable"

	// Enable print errors and warnings
	Enable VerboseLevel = "Enable"

	// Moar print client commands and responses
	Moar VerboseLevel = "Moar"

	// Extreme print internal state transactions
	Extreme VerboseLevel = "Extreme"
)

// MemcachedImage struct for storing image and tag for Memcached
type MemcachedImage struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// ===============================================================================
// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
	// Size defines the number of Memcached instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	Size int32 `json:"size,omitempty"`

	// Port defines the port that will be used to init the container with the image
	ContainerPort int32 `json:"containerPort,omitempty"`

	// This flag tells the controller to use or not Twemproxy.
	// +optional
	Proxy *bool `json:"proxy,omitempty"`

	// Parameter for setting image and tag for memcached pod
	// default 'memcached:1.6.23-alpine'
	// +optional
	Image MemcachedImage `json:"image"`

	// Specifies the verbose level.
	// Valid values are:
	// - "Disable": no verbose output at all;
	// - "Enable"(default): print errors and warnings;
	// - "Moar": print client commands and responses;
	// - "Extreme": print internal state transactions;
	// +optional
	Verbose VerboseLevel `json:"verbose,omitempty"`

	// Resources defines CPU and memory for Memcached prods
	*Resources `json:"resources,omitempty"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	// Represents the observations of a Memcached's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// ===============================================================================
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Memcached is the Schema for the memcacheds API
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MemcachedList contains a list of Memcached
type MemcachedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Memcached `json:"items"`
}

// ===============================================================================
func init() {
	SchemeBuilder.Register(&Memcached{}, &MemcachedList{})
}
