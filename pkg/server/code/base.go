package code

// Common: basic errors.
const (
	// ErrSuccess - 200: OK.
	ErrSuccess int = iota + 100001

	// ErrUnknown - 500: Internal server error.
	ErrUnknown

	// ErrBind - 400: Error occurred while binding the request body to the struct.
	ErrBind
)

func init() {
	register(ErrDashboardNotFound, 404, "Dashboards not found")
	register(ErrSuccess, 200, "OK")
	register(ErrClusterNotFound, 404, "Cluster not found")
	register(ErrClusterDuplicated, 400, "Cluster duplicated")
	register(ErrClusterAdd, 500, "Cluster add failed")
	register(ErrClusterUpdate, 500, "Cluster update failed")
	register(ErrClusterDelete, 500, "Cluster delete failed")
	register(ErrUnknown, 500, "Internal server error")
	register(ErrBind, 400, "Error occurred while binding the request body to the struct")

}
