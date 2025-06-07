package zmysql

type IS_LIST_TYPE int8

const (
	HAS_ONE IS_LIST_TYPE = iota + 1 // 有一个
	HAS_LIST
)
