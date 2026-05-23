package handler

// Handler unified handler container
type Handler struct {
	Employee   *EmployeeHandler
	Department *DepartmentHandler
	Product    *ProductHandler
}

// NewHandler creates Handler container
func NewHandler(emp *EmployeeHandler, dept *DepartmentHandler, prod *ProductHandler) *Handler {
	return &Handler{
		Employee:   emp,
		Department: dept,
		Product:    prod,
	}
}
