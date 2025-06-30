package zmysql

import (
	_ "github.com/go-sql-driver/mysql"
)

// -------------------------  下面是exec  -----------------------

// 执行查询并返回是否成功
func Exec(query string, args ...any) (bool, error) {
	return mysql_client.Exec(query, args...)
}

// ExecByte 执行原生 SQL 查询并返回结果集的所有数据，格式为 []byte
func ExecByte(query string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	return mysql_client.ExecByte(query, isList, args...)
}

// ExecProcByte 执行存储过程并返回结果集的所有数据，格式为 []byte
func ExecProcByte(procName string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	return mysql_client.ExecProcByte(procName, isList, args...)
}
