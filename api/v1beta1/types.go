package v1beta1

// ServerStatus represents the status of subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusPending   = SubscriptionStatus("Pending")
	SubscriptionStatusActive    = SubscriptionStatus("Active")
	SubscriptionStatusSuspended = SubscriptionStatus("Suspended")
	SubscriptionStatusClosed    = SubscriptionStatus("Closed")
)

// PowerStatus represents that the VPS is powerd on or not
type PowerStatus string

const (
	PowerStatusStarting = PowerStatus("Starting")
	PowerStatusStopped  = PowerStatus("Stopped")
	PowerStatusRunning  = PowerStatus("Running")
)

// ServerState represents a detail of server state.
type ServerState string

const (
	ServerStateNone        = ServerState("None")
	ServerStateLocked      = ServerState("Locked")
	ServerStateBooting     = ServerState("Booting")
	ServerStateIsoMounting = ServerState("IsoMounting")
	ServerStateOK          = ServerState("OK")
	ServerStateError       = ServerState("Error")
)
