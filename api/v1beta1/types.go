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

// ServerStatus represents the status of server.
type ServerStatus string

const (
	ServerStatusNone              = ServerStatus("None")
	ServerStatusLocked            = ServerStatus("Locked")
	ServerStatusInstallingBooting = ServerStatus("InstallingBooting")
	ServerStatusOK                = ServerStatus("OK")
	ServerStatusError             = ServerStatus("Error")
)

// SnapshotStatus represents the status of snapshot.
type SnapshotStatus string

const (
	SnapshotStatusNone     = SnapshotStatus("None")
	SnapshotStatusPending  = SnapshotStatus("Pending")
	SnapshotStatusComplete = SnapshotStatus("Complete")
	SnapshotStatusDeleted  = SnapshotStatus("Deleted")
	SnapshotStatusError    = SnapshotStatus("Error")
)
