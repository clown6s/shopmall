package model

import (
	"time"
)

// Category 商品三级分类
type Category struct {
	CatID      int64   `json:"catId"`
	Name       string  `json:"name"`
	ParentCid  int64   `json:"parentCid"`
	CatLevel   int     `json:"catLevel"`
	ShowStatus int8    `json:"showStatus"`
	Sort       int     `json:"sort"`
	Icon       *string `json:"icon"`
}

// Brand 品牌
type Brand struct {
	BrandID     int64  `json:"brandId"`
	CatId       int64  `json:"catId"`
	Name        string `json:"name"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
	ShowStatus  *int8  `json:"showStatus"`
	FirstLetter string `json:"firstLetter"`
	Sort        int    `json:"sort"`
}

// Attr 商品属性
type Attr struct {
	AttrID      int64  `db:"attr_id"       json:"attrId"`
	AttrName    string `db:"attr_name"     json:"attrName"`
	SearchType  int8   `db:"search_type"   json:"searchType"`
	Icon        string `db:"icon"          json:"icon"`
	ValueSelect string `db:"value_select" json:"valueSelect"`
	AttrType    int8   `db:"attr_type"     json:"attrType"`
	Enable      int64  `db:"enable"        json:"enable"`
	CatelogID   int64  `db:"catelog_id"   json:"catelogId"`
	ShowDesc    int8   `db:"show_desc"     json:"showDesc"`
}

// AttrGroup 属性分组
type AttrGroup struct {
	AttrGroupID   int64  `db:"attr_group_id"   json:"attrGroupId"`
	AttrGroupName string `db:"attr_group_name" json:"attrGroupName"`
	Sort          int    `db:"sort"             json:"sort"`
	Descript      string `db:"descript"         json:"descript"`
	Icon          string `db:"icon"             json:"icon"`
	CatelogID     int64  `db:"catelog_id"       json:"catelogId"`
}

// AttrAttrgroupRelation 属性&属性分组关联
type AttrAttrgroupRelation struct {
	ID          int64 `db:"id"             json:"id"`
	AttrID      int64 `db:"attr_id"        json:"attrId"`
	AttrGroupID int64 `db:"attr_group_id"  json:"attrGroupId"`
	AttrSort    int   `db:"attr_sort"      json:"attrSort"`
}

// SpuInfo spu信息
type SpuInfo struct {
	ID             int64     `db:"id"                json:"id"`
	SpuName        string    `db:"spu_name"          json:"spuName"`
	SpuDescription string    `db:"spu_description"   json:"spuDescription"`
	CatalogID      int64     `db:"catalog_id"        json:"catalogId"`
	BrandID        int64     `db:"brand_id"          json:"brandId"`
	Weight         float64   `db:"weight"            json:"weight"`
	PublishStatus  int8      `db:"publish_status"    json:"publishStatus"`
	CreateTime     time.Time `db:"create_time"       json:"createTime"`
	UpdateTime     time.Time `db:"update_time"       json:"updateTime"`
}

// SpuInfoDesc spu信息介绍
type SpuInfoDesc struct {
	SpuID   int64  `db:"spu_id"   json:"spuId"`
	Decript string `db:"decript"  json:"decript"`
}

// SpuImages spu图片
type SpuImages struct {
	ID         int64  `db:"id"          json:"id"`
	SpuID      int64  `db:"spu_id"      json:"spuId"`
	ImgName    string `db:"img_name"    json:"imgName"`
	ImgUrl     string `db:"img_url"     json:"imgUrl"`
	ImgSort    int    `db:"img_sort"    json:"imgSort"`
	DefaultImg int8   `db:"default_img" json:"defaultImg"`
}

// SpuComment 商品评价
type SpuComment struct {
	ID             int64     `db:"id"               json:"id"`
	SkuID          int64     `db:"sku_id"           json:"skuId"`
	SpuID          int64     `db:"spu_id"           json:"spuId"`
	SpuName        string    `db:"spu_name"         json:"spuName"`
	MemberNickName string    `db:"member_nick_name" json:"memberNickName"`
	Star           int8      `db:"star"             json:"star"`
	MemberIP       string    `db:"member_ip"        json:"memberIp"`
	CreateTime     time.Time `db:"create_time"      json:"createTime"`
	ShowStatus     int8      `db:"show_status"      json:"showStatus"`
	SpuAttributes  string    `db:"spu_attributes"   json:"spuAttributes"`
	LikesCount     int       `db:"likes_count"      json:"likesCount"`
	ReplyCount     int       `db:"reply_count"      json:"replyCount"`
	Resources      string    `db:"resources"        json:"resources"`
	Content        string    `db:"content"          json:"content"`
	MemberIcon     string    `db:"member_icon"      json:"memberIcon"`
	CommentType    int8      `db:"comment_type"     json:"commentType"`
}

// CommentReplay 商品评价回复关系
type CommentReplay struct {
	ID        int64 `db:"id"          json:"id"`
	CommentID int64 `db:"comment_id" json:"commentId"`
	ReplyID   int64 `db:"reply_id"   json:"replyId"`
}

// SkuInfo sku信息
type SkuInfo struct {
	SkuID         int64   `db:"sku_id"             json:"skuId"`
	SpuID         int64   `db:"spu_id"             json:"spuId"`
	SkuName       string  `db:"sku_name"           json:"skuName"`
	SkuDesc       string  `db:"sku_desc"           json:"skuDesc"`
	CatalogID     int64   `db:"catalog_id"         json:"catalogId"`
	BrandID       int64   `db:"brand_id"           json:"brandId"`
	SkuDefaultImg string  `db:"sku_default_img"   json:"skuDefaultImg"`
	SkuTitle      string  `db:"sku_title"          json:"skuTitle"`
	SkuSubtitle   string  `db:"sku_subtitle"       json:"skuSubtitle"`
	Price         float64 `db:"price"              json:"price"`
	SaleCount     int64   `db:"sale_count"         json:"saleCount"`
}

// SkuImages sku图片
type SkuImages struct {
	ID         int64  `db:"id"          json:"id"`
	SkuID      int64  `db:"sku_id"      json:"skuId"`
	ImgUrl     string `db:"img_url"     json:"imgUrl"`
	ImgSort    int    `db:"img_sort"    json:"imgSort"`
	DefaultImg int    `db:"default_img" json:"defaultImg"`
}

// SkuSaleAttrValue sku销售属性&值
type SkuSaleAttrValue struct {
	ID        int64  `db:"id"          json:"id"`
	SkuID     int64  `db:"sku_id"      json:"skuId"`
	AttrID    int64  `db:"attr_id"     json:"attrId"`
	AttrName  string `db:"attr_name"   json:"attrName"`
	AttrValue string `db:"attr_value"   json:"attrValue"`
	AttrSort  int    `db:"attr_sort"   json:"attrSort"`
}

// ProductAttrValue spu属性值
type ProductAttrValue struct {
	ID        int64  `db:"id"           json:"id"`
	SpuID     int64  `db:"spu_id"       json:"spuId"`
	AttrID    int64  `db:"attr_id"      json:"attrId"`
	AttrName  string `db:"attr_name"    json:"attrName"`
	AttrValue string `db:"attr_value"   json:"attrValue"`
	AttrSort  int    `db:"attr_sort"    json:"attrSort"`
	QuickShow int8   `db:"quick_show"   json:"quickShow"`
}
