package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// KrumkakeClusterTemplateSpec defines the desired state of KrumkakeClusterTemplate
type KrumkakeClusterTemplateSpec struct {
	Template KrumkakeClusterTemplateResource `json:"template"`
}

// KrumkakeClusterTemplateResource contains spec for KrumkakeClusterSpec.
type KrumkakeClusterTemplateResource struct {
	ObjectMeta clusterv1beta2.ObjectMeta `json:"metadata,omitempty"`
	Spec       KrumkakeClusterSpec       `json:"spec"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=krumkakeclustertemplates,scope=Namespaced,categories=cluster-api,shortName=kct

// KrumkakeClusterTemplate is the Schema for the krumkakeclustertemplates API
type KrumkakeClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KrumkakeClusterTemplateSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// KrumkakeClusterTemplateList contains a list of KrumkakeClusterTemplate
type KrumkakeClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrumkakeClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeClusterTemplate{}, &KrumkakeClusterTemplateList{})
}
