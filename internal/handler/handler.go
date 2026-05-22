package handler

// Handler unified handler container
type Handler struct {
	// Product    *ProductHandler
	// Employee   *EmployeeHandler
	// Department *DepartmentHandler
	// Role       *RoleHandler
	// Permission *PermissionHandler
}

// NewHandler creates Handler container
func NewHandler() *Handler {
	return &Handler{}
}
