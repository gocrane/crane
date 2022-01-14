package code

// crane-server: errors.
const (
	// ErrDashboardNotFound - 404: Dashboards not found.
	ErrDashboardNotFound int = iota + 110001
)

const (
	ErrClusterNotFound int = iota + 110101
	ErrClusterDelete
	ErrClusterDuplicated
	ErrClusterAdd
	ErrClusterUpdate
)
