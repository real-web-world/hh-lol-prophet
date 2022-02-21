package fastcurd

type RespJsonExtra struct {
	ReqID    string      `json:"requestID,omitempty"`
	SQLs     interface{} `json:"sqls,omitempty"`
	ProcTime string      `json:"procTime" example:"0.2s"`
	TempData interface{} `json:"tempData,omitempty"`
}

// 通用返回json
// 所有的接口均返回此对象
type RetJSON struct {
	Code  int            `json:"code" example:"0"`
	Data  interface{}    `json:"data,omitempty"`
	Msg   string         `json:"msg,omitempty" example:"提示信息"`
	Count *int           `json:"count,omitempty"`
	Page  int            `json:"page,omitempty"`
	Limit int            `json:"limit,omitempty"`
	Extra *RespJsonExtra `json:"extra,omitempty"`
}
