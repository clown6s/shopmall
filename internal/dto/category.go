package dto

// CascaderNode 分类级联节点
type CascaderNode struct {
	CatID      int64           `json:"catId"`
	Name       string          `json:"name"`
	CatLevel   int             `json:"catLevel"`
	Sort       int             `json:"sort"`
	ShowStatus int8            `json:"showStatus"`
	Children   []*CascaderNode `json:"children,omitempty"`
}

// CategoryEditRequest 分类编辑请求
type CategoryEditRequest struct {
	CatID      int64  `json:"catId"`
	Name       string `json:"name"`
	Sort       int    `json:"sort"`
	ShowStatus int8   `json:"showStatus"`
}

// CategoryAddRequest 分类新增请求
type CategoryAddRequest struct {
	Name       string `json:"name"`
	ParentCid  int64  `json:"parentCid"`
	CatLevel   int    `json:"catLevel"`
	ShowStatus int8   `json:"showStatus"`
	Sort       int    `json:"sort"`
	Icon       string `json:"icon,omitempty"`
}
