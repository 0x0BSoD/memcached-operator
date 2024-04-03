package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

// DockerImage struct for storing image and tag for docker image
type DockerImage struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ProgressState string

const (
	DefaultPort                         = 11211
	Finalizer                           = "cache.bsod.io/finalizer"
	NoFinalizerAnnotation               = "cache.bsod.io/no-finalizer"
	ProgressUpdating      ProgressState = "Updating"
	ProgressReady         ProgressState = "Ready"
	MemcachedLabel                      = "cache.bsod.io/memcached"
)

// ===============================================================================
// MemcachedSpec defines the desired state of Memcached
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:XPreserveUnknownFields
type MemcachedSpec struct {
	// Size defines the number of Memcached instances
	// +kubebuilder:validation:Minimum=1
	Size int32 `json:"size,omitempty"`

	// Port defines the port that will be used to init the container with the image
	ContainerPort int32 `json:"containerPort,omitempty"`

	// Parameter for setting image and tag for memcached pod
	// default 'memcached:1.6.23-alpine'
	// +optional
	Image DockerImage `json:"image,omitempty"`

	// Resources defines CPU and memory for Memcached pods
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// ===============================================================================
// Condition
type MemcachedConditionType string

const (
	MemcachedReady       MemcachedConditionType = "Ready"
	MemcachedDegraded    MemcachedConditionType = "Degraded"
	MemcacheDecommission MemcachedConditionType = "Decommission"
	MemcachedScalingUp   MemcachedConditionType = "ScalingUp"
	MemcachedScalingDown MemcachedConditionType = "ScalingDown"
	MemcachedUpdating    MemcachedConditionType = "Updating"
)

type MemcachedCondition struct {
	Type               MemcachedConditionType `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	// Represents the observations of a Memcached's current state.
	Conditions []MemcachedCondition `json:"conditions,omitempty"`
	// Last known progress state
	// +optional
	OperatorProgress ProgressState `json:"operatorProgress,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ===============================================================================
// Memcached is the Schema for the memcacheds API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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

// ===============================================================================
func (m *Memcached) GetCondition(conditionType MemcachedConditionType) (MemcachedCondition, bool) {
	for _, condition := range m.Status.Conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}

	return MemcachedCondition{}, false
}

func (status *MemcachedStatus) GetConditionStatus(conditionType MemcachedConditionType) corev1.ConditionStatus {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}
