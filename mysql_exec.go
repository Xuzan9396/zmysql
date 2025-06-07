package zmysql

import (
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strings"
)

// -------------------------  下面是exec  -----------------------

// 执行查询并返回是否成功
func Exec(query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %v", err)
	}

	return rowsAffected > 0, nil
}

// ExecByte 执行原生 SQL 查询并返回结果集的所有数据，格式为 []byte
func ExecByte(query string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	mysql_client.debugLog(query, args...)
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 用于存储结果集的所有数据
	var resultData []map[string]any

	// 遍历每一行
	for rows.Next() {
		// 创建一个存储每列数据的容器
		rowData := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowData {
			rowPointers[i] = &rowData[i]
		}

		// 扫描当前行数据
		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 将当前行数据转换为 map[string]any
		rowMap := make(map[string]any)
		for i, col := range columns {
			rowMap[col] = rowData[i]
		}

		// 添加到结果集
		resultData = append(resultData, rowMap)
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	// 如果只有一条数据，直接返回该数据的 JSON
	if isList == HAS_ONE && len(resultData) >= 1 {
		jsonData, err := json.Marshal(resultData[0])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal single result data: %v", err)
		}
		return jsonData, nil
	}

	// 如果有多条数据，返回 JSON 切片
	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result data: %v", err)
	}

	return jsonData, nil
}

// ExecProcByte 执行存储过程并返回结果集的所有数据，格式为 []byte
func ExecProcByte(procName string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	mysql_client.debugLog(procName, args...)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 用于存储结果集的所有数据
	var resultData []map[string]any

	// 遍历当前结果集的每一行
	for rows.Next() {
		// 创建一个存储每列数据的容器
		rowData := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowData {
			rowPointers[i] = &rowData[i]
		}

		// 扫描当前行数据
		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 将当前行数据转换为 map[string]any
		rowMap := make(map[string]any)
		for i, col := range columns {
			rowMap[col] = rowData[i]
		}

		// 添加到结果集
		resultData = append(resultData, rowMap)
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	// 如果只有一条数据，直接返回该数据的 JSON
	if isList == HAS_ONE && len(resultData) >= 1 {
		jsonData, err := json.Marshal(resultData[0])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal single result data: %v", err)
		}
		return jsonData, nil
	}

	// 如果有多条数据，返回 JSON 切片
	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result data: %v", err)
	}

	return jsonData, nil
}
