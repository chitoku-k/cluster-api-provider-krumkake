package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// KrumkakeMachineTemplateSpec defines the desired state of KrumkakeMachineTemplate
type KrumkakeMachineTemplateSpec struct {
	Template KrumkakeMachineTemplateResource `json:"template"`
}

// KrumkakeMachineTemplateResource contains spec for KrumkakeMachineSpec.
type KrumkakeMachineTemplateResource struct {
	ObjectMeta clusterv1beta2.ObjectMeta `json:"metadata,omitempty"`
	Spec       KrumkakeMachineSpec       `json:"spec"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=krumkakemachinetemplates,scope=Namespaced,categories=cluster-api,shortName=kmt

// KrumkakeMachineTemplate is the Schema for the krumkakemachinetemplates API
type KrumkakeMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec KrumkakeMachineTemplateSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// KrumkakeMachineTemplateList contains a list of KrumkakeMachineTemplate
type KrumkakeMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []KrumkakeMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeMachineTemplate{}, &KrumkakeMachineTemplateList{})
}
