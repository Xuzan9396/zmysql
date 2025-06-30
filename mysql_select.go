package zmysql

import "github.com/Xuzan9396/zmysql/smysql"

// Find 执行查询并将结果映射到结构体中 列表查询
func Find(dest any, query string, args ...any) error {
	return mysql_client.Find(dest, query, args...)
}

// FindProc 执行存储过程并将结果映射到结构体中 列表查询
func FindProc(dest any, procName string, args ...any) error {
	return mysql_client.FindProc(dest, procName, args...)
}

// First 执行查询并将结果映射到结构体中，查询一条数据
func First(dest any, query string, args ...any) (bool, error) {
	return mysql_client.First(dest, query, args...)
}

// FirstProc 执行存储过程并将结果映射到结构体中，查询一条数据
func FirstProc(dest any, procName string, args ...any) (bool, error) {
	return mysql_client.FirstProc(dest, procName, args...)
}

// FirstCol 执行查询并将单个字段值映射到基础类型
func FirstCol(dest any, query string, args ...any) (bool, error) {
	return mysql_client.FirstCol(dest, query, args...)
}

// FirstColProc 执行存储过程并将单个字段值映射到基础类型
func FirstColProc(dest any, procName string, args ...any) (bool, error) {
	return mysql_client.FirstColProc(dest, procName, args...)
}

// FindMultipleProc 执行存储过程并将多个结果集映射到多个目标结构体或切片中
func FindMultipleProc(dest []any, procName string, args ...any) error {
	return mysql_client.FindMultipleProc(dest, procName, args...)
}

// ExecFindLastId 执行SQL查询并返回LastInsertId
func ExecFindLastId(query string, args ...any) (int64, error) {
	return mysql_client.ExecFindLastId(query, args...)
}

// FindArray 执行查询并返回指定字段的泛型数组
func FindArray[T int64 | string](fieldName string, query string, args ...any) ([]T, error) {
	return smysql.FindArray[T](mysql_client, fieldName, query, args...)
}

// FindArrayInt64 执行查询并返回指定字段的int64数组
func FindArrayInt64(fieldName string, query string, args ...any) ([]int64, error) {
	return mysql_client.FindArrayInt64(fieldName, query, args...)
}

// FindArrayString 执行查询并返回指定字段的string数组
func FindArrayString(fieldName string, query string, args ...any) ([]string, error) {
	return mysql_client.FindArrayString(fieldName, query, args...)
}

// FindProcArray 执行存储过程并返回指定字段的泛型数组
func FindProcArray[T int64 | string](fieldName string, procName string, args ...any) ([]T, error) {
	return smysql.FindProcArray[T](mysql_client, fieldName, procName, args...)
}

// FindProcArrayInt64 执行存储过程并返回指定字段的int64数组
func FindProcArrayInt64(fieldName string, procName string, args ...any) ([]int64, error) {
	return mysql_client.FindProcArrayInt64(fieldName, procName, args...)
}

// FindProcArrayString 执行存储过程并返回指定字段的string数组
func FindProcArrayString(fieldName string, procName string, args ...any) ([]string, error) {
	return mysql_client.FindProcArrayString(fieldName, procName, args...)
}

// FindMap 执行查询并返回泛型键值对映射
func FindMap[T comparable, Y any](keyField string, valueField string, query string, args ...any) (map[T]Y, error) {
	return smysql.FindMap[T, Y](mysql_client, keyField, valueField, query, args...)
}

// FindProcMap 执行存储过程并返回泛型键值对映射
func FindProcMap[T comparable, Y any](keyField string, valueField string, procName string, args ...any) (map[T]Y, error) {
	return smysql.FindProcMap[T, Y](mysql_client, keyField, valueField, procName, args...)
}

// FirstColAny 执行查询并返回指定类型的单列值（泛型版本）
func FirstColAny[T int64 | string](query string, args ...any) (T, bool, error) {
	return smysql.FirstColAny[T](mysql_client, query, args...)
}

// FirstColProcAny 执行存储过程并返回指定类型的单列值（泛型版本）
func FirstColProcAny[T int64 | string](procName string, args ...any) (T, bool, error) {
	return smysql.FirstColProcAny[T](mysql_client, procName, args...)
}

// FirstColInt64 执行查询并返回int64类型的单列值
func FirstColInt64(query string, args ...any) (int64, bool, error) {
	return mysql_client.FirstColInt64(query, args...)
}

// FirstColString 执行查询并返回string类型的单列值
func FirstColString(query string, args ...any) (string, bool, error) {
	return mysql_client.FirstColString(query, args...)
}

// FirstColProcInt64 执行存储过程并返回int64类型的单列值
func FirstColProcInt64(procName string, args ...any) (int64, bool, error) {
	return mysql_client.FirstColProcInt64(procName, args...)
}

// FirstColProcString 执行存储过程并返回string类型的单列值
func FirstColProcString(procName string, args ...any) (string, bool, error) {
	return mysql_client.FirstColProcString(procName, args...)
}
