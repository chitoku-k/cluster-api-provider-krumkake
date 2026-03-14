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
	ProviderID *string `json:"providerID,omitempty"`

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

	// The snapshot_id to use when deploying this instance.
	SnapshotID string `json:"snapshotID,omitempty"`

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
	// Ready represents that the infrastructure is ready.
	// +kubebuilder:default:=false
	Ready bool `json:"ready"`

	// CPU represents the number of virtual CPUs of the machine.
	// +optional
	CPU int `json:"cpu,omitempty"`

	// RAM represents the amount of memory in MB of the machine.
	// +optional
	RAM int `json:"ram,omitempty"`

	// Storage represents the disk size in GB of the machine.
	// +optional
	Storage int `json:"storage,omitempty"`

	// Addresses contains the associated addresses for the machine.
	Addresses []clusterv1beta2.MachineAddress `json:"addresses"`

	// Vultr represents the Vultr machine's status.
	Vultr *KrumkakeMachineVultrStatus `json:"vultr,omitempty"`

	// Conditions defines current service state of the KrumkakeMachine.
	// +optional
	Conditions clusterv1beta2.Conditions `json:"conditions,omitempty"`
}

// KrumkakeMachineVultrStatus defines the observed Vultr's state of KrumkakeMachine.
type KrumkakeMachineVultrStatus struct {
	// ServerStatus represents the status of subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// PowerStatus represents that the VPS is powerd on or not
	// +optional
	PowerStatus *PowerStatus `json:"powerStatus,omitempty"`

	// ServerState represents a detail of server state.
	// +optional
	ServerState *ServerState `json:"serverState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this KrumkakeMachine belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
// +kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="Instance ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns this KrumkakeMachine"
// +kubebuilder:resource:path=krumkakemachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status

// KrumkakeMachine is the Schema for the krumkakemachines API.
type KrumkakeMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   KrumkakeMachineSpec   `json:"spec"`
	Status KrumkakeMachineStatus `json:"status,omitzero"`
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
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []KrumkakeMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeMachine{}, &KrumkakeMachineList{})
}
