package handler

// Handler unified handler container
type Handler struct {
	Employee   *EmployeeHandler
	Department *DepartmentHandler
	Product    *ProductHandler
	Role       *RoleHandler
	Permission *PermissionHandler
}

// NewHandler creates Handler container
func NewHandler(emp *EmployeeHandler, dept *DepartmentHandler, prod *ProductHandler, role *RoleHandler, perm *PermissionHandler) *Handler {
	return &Handler{
		Employee:   emp,
		Department: dept,
		Product:    prod,
		Role:       role,
		Permission: perm,
	}
}
