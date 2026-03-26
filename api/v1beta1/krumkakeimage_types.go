package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	// ImageFinalizer allows KrumkakeImageReconciler to clean up resources associated with KrumkakeImage before removing it from the apiserver.
	ImageFinalizer = "krumkakeimage.infrastructure.cluster.x-k8s.io"
)

// KrumkakeImageSpec defines the desired state of KrumkakeImage.
type KrumkakeImageSpec struct {
	// OSImage defines the operating system of the image.
	OSImage string `json:"osImage"`

	// Version defines the Kubernetes version of the image.
	Version string `json:"version"`

	// UEFI defines if the image is UEFI.
	UEFI bool `json:"uefi"`

	// URL defines the URL of the raw image.
	URL string `json:"url"`
}

// KrumkakeImageStatus defines the observed state of KrumkakeImage.
type KrumkakeImageStatus struct {
	// Vultr represents the Vultr snapshot's status.
	// +optional
	Vultr KrumkakeImageVultrStatus `json:"vultr,omitempty,omitzero"`

	// Conditions defines current service state of the KrumkakeImage.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// KrumkakeImageVultrStatus defines the observed state of KrumkakeImage.
type KrumkakeImageVultrStatus struct {
	// SnapshotID represents the ID of the snapshot.
	// +optional
	SnapshotID string `json:"snapshotID,omitempty"`

	// SnapshotStatus represents the status of the snapshot.
	// +optional
	SnapshotStatus *SnapshotStatus `json:"snapshotStatus,omitempty"`
}

func (k *KrumkakeImageVultrStatus) GetSnapshotID() string {
	return k.SnapshotID
}

func (k *KrumkakeImageVultrStatus) GetSnapshotStatus() SnapshotStatus {
	return ptr.Deref(k.SnapshotStatus, SnapshotStatusNone)
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Vultr Status",type="string",JSONPath=".status.vultr.snapshotStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=krumkakeimages,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status

// KrumkakeImage is the Schema for the krumkakeimages API.
type KrumkakeImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrumkakeImageSpec   `json:"spec,omitempty"`
	Status KrumkakeImageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrumkakeImageList contains a list of KrumkakeImage.
type KrumkakeImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []KrumkakeImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrumkakeImage{}, &KrumkakeImageList{})
}
