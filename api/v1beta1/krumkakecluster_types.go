package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// ClusterFinalizer allows KrumkakeClusterReconciler to clean up resources associated with KrumkakeCluster before removing it from the apiserver.
	ClusterFinalizer = "krumkakecluster.infrastructure.cluster.x-k8s.io"
)

// KrumkakeClusterSpec defines the desired state of KrumkakeCluster.
type KrumkakeClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1beta2.APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`
}

// KrumkakeClusterStatus defines the observed state of KrumkakeCluster.
type KrumkakeClusterStatus struct {
	// Initialization represents the observations of the KrumkakeCluster initialization process.
	// +optional
	Initialization KrumkakeClusterInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Conditions defines current service state of the KrumkakeCluster.
	// +optional
	Conditions clusterv1beta2.Conditions `json:"conditions,omitempty"`
}

// KrumkakeClusterInitializationStatus defines the initialization status of KrumkakeCluster.
type KrumkakeClusterInitializationStatus struct {
	// Provisioned represents whether the infrastructure is fully provisioned.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this KrumkakeCluster belongs"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=krumkakeclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status

// KrumkakeCluster is the Schema for the krumkakeclusters API
type KrumkakeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrumkakeClusterSpec   `json:"spec,omitempty"`
	Status KrumkakeClusterStatus `json:"status,omitempty"`
}

func (k *KrumkakeCluster) GetConditions() clusterv1beta2.Conditions {
	return k.Status.Conditions
}

func (k *KrumkakeCluster) SetConditions(conditions clusterv1beta2.Conditions) {
	k.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// KrumkakeClusterList contains a list of KrumkakeCluster.
type KrumkakeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrumkakeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeCluster{}, &KrumkakeClusterList{})
}
