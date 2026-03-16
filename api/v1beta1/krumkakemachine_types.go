package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// MachineFinalizer allows KrumkakeMachineReconciler to clean up resources associated with KrumkakeMachine before removing it from the apiserver.
	MachineFinalizer = "krumkakemachine.infrastructure.cluster.x-k8s.io"
)

// KrumkakeMachineSpec defines the desired state of KrumkakeMachine.
type KrumkakeMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID string `json:"providerID,omitempty"`

	// ImageName is the name of the KrumkakeImage.
	// +kubebuilder:validation:Required
	ImageName string `json:"imageName"`

	// Vultr is the spec of the Vultr machine.
	// +optional
	Vultr KrumkakeMachineVultrSpec `json:"vultr"`
}

// KrumkakeMachineVultrSpec defines the desired Vultr's state of KrumkakeMachineVultrSpec.
type KrumkakeMachineVultrSpec struct {
	// The Vultr Region (DCID) the machine lives on.
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// PlanID is the ID of the Vultr VPS plan.
	PlanID string `json:"planID,omitempty"`

	// VPCID is the ID of the VPC to be attached.
	// +optional
	VPCID string `json:"vpcID,omitempty"`

	// VPCOnly indicates that the VPS will not receive a public IP or public NIC when true.
	VPCOnly bool `json:"vpcOnly,omitempty"`

	// The Vultr firewall group ID to attach to the instance.
	// +optional
	FirewallGroupID string `json:"firewallGroupID,omitempty"`

	// SSHKeys is the list of the SSH keys to attach to the instance.
	// +optional
	SSHKeys []string `json:"sshKeys,omitempty"`
}

// KrumkakeMachineStatus defines the observed state of KrumkakeMachine.
type KrumkakeMachineStatus struct {
	// Initialization represents the observations of the KrumkakeMachine initialization process.
	// +optional
	Initialization KrumkakeMachineInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Addresses contains the associated addresses for the machine.
	// +optional
	Addresses []clusterv1beta2.MachineAddress `json:"addresses"`

	// CPU represents the number of virtual CPUs of the machine.
	// +optional
	CPU int `json:"cpu,omitempty"`

	// RAM represents the amount of memory in MB of the machine.
	// +optional
	RAM int `json:"ram,omitempty"`

	// Storage represents the disk size in GB of the machine.
	// +optional
	Storage int `json:"storage,omitempty"`

	// Vultr represents the Vultr machine's status.
	// +optional
	Vultr *KrumkakeMachineVultrStatus `json:"vultr,omitempty"`

	// Conditions defines current service state of the KrumkakeMachine.
	// +optional
	Conditions clusterv1beta2.Conditions `json:"conditions,omitempty"`
}

// KrumkakeMachineInitializationStatus defines the initialization status of KrumkakeMachine.
type KrumkakeMachineInitializationStatus struct {
	// Provisioned represents whether the infrastructure is fully provisioned.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// KrumkakeMachineVultrStatus defines the observed Vultr's state of KrumkakeMachine.
type KrumkakeMachineVultrStatus struct {
	// ServerStatus represents the status of subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// PowerStatus represents the power status of server.
	// +optional
	PowerStatus *PowerStatus `json:"powerStatus,omitempty"`

	// ServerState represents the state of server.
	// +optional
	ServerState *ServerState `json:"serverState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this KrumkakeMachine belongs"
// +kubebuilder:printcolumn:name="Provider ID",type="string",JSONPath=".spec.providerID",description="Provider ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns this KrumkakeMachine"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=krumkakemachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status

// KrumkakeMachine is the Schema for the krumkakemachines API.
type KrumkakeMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrumkakeMachineSpec   `json:"spec"`
	Status KrumkakeMachineStatus `json:"status,omitempty"`
}

func (k *KrumkakeMachine) GetConditions() clusterv1beta2.Conditions {
	return k.Status.Conditions
}

func (k *KrumkakeMachine) SetConditions(conditions clusterv1beta2.Conditions) {
	k.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// KrumkakeMachineList contains a list of KrumkakeMachine.
type KrumkakeMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrumkakeMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeMachine{}, &KrumkakeMachineList{})
}
