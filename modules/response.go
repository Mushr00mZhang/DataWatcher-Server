package modules

type MyResponse[T any] struct {
	Result T      // 结果
	Tip    string // 提示信息
	Error  string // 错误信息
}

type PagedList[T any] struct {
	Items []T   // 列表
	Total int32 // 总数
}
