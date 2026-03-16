package v1beta1

// ServerStatus represents the status of subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusPending   = SubscriptionStatus("Pending")
	SubscriptionStatusActive    = SubscriptionStatus("Active")
	SubscriptionStatusSuspended = SubscriptionStatus("Suspended")
	SubscriptionStatusClosed    = SubscriptionStatus("Closed")
)

// PowerStatus represents the power status of server.
type PowerStatus string

const (
	PowerStatusStarting = PowerStatus("Starting")
	PowerStatusStopped  = PowerStatus("Stopped")
	PowerStatusRunning  = PowerStatus("Running")
)

// ServerState represents the state of server.
type ServerState string

const (
	ServerStateNone        = ServerState("None")
	ServerStateLocked      = ServerState("Locked")
	ServerStateBooting     = ServerState("Booting")
	ServerStateIsoMounting = ServerState("IsoMounting")
	ServerStateOK          = ServerState("OK")
	ServerStateError       = ServerState("Error")
)

// SnapshotState represents the state of snapshot.
type SnapshotState string

const (
	SnapshotStateNone     = SnapshotState("None")
	SnapshotStatePending  = SnapshotState("Pending")
	SnapshotStateComplete = SnapshotState("Complete")
	SnapshotStateDeleted  = SnapshotState("Deleted")
	SnapshotStateError    = SnapshotState("Error")
)
