package zmysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Xuzan9396/zlog"
	"log"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLClient 用于 MySQL 客户端
type MySQLClient struct {
	DB *sql.DB
	// 可选的连接池配置
	connMaxLifetime time.Duration
	maxOpenConns    int
	maxIdleConns    int
	loc             string
	debug           bool

	mu     sync.RWMutex
	fields map[reflect.Type]map[string]int // Type -> {column name -> field index}
}

var mysql_client *MySQLClient

// Conn 创建并初始化一个新的 MySQL 客户端
func Conn(username, password, addr, dbName string, opts ...func(*MySQLClient)) error {
	client := &MySQLClient{
		connMaxLifetime: 4 * time.Hour, // 默认连接最大生命周期
		maxOpenConns:    100,           // 默认最大连接数
		maxIdleConns:    50,            // 默认最大空闲连接数
		fields:          make(map[reflect.Type]map[string]int),
		loc:             url.QueryEscape("Local"),
	}

	// 应用可选的配置选项
	for _, opt := range opts {
		opt(client)
	}

	// URL 编码用户名和密码
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true&loc=%s", username, password, addr, dbName, client.loc)

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// 设置连接池参数
	db.SetConnMaxLifetime(client.connMaxLifetime)
	db.SetMaxOpenConns(client.maxOpenConns)
	db.SetMaxIdleConns(client.maxIdleConns)

	// 检查数据库连接
	err = db.Ping()
	if err != nil {
		log.Fatalf("数据库连接失败ping:%v", err)
	}
	client.DB = db
	mysql_client = client

	return nil
}

// 设置连接最大生命周期
func WithConnMaxLifetime(d time.Duration) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.connMaxLifetime = d
	}
}

// 设置最大连接数
func WithMaxOpenConns(n int) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.maxOpenConns = n
	}
}

// 设置最大空闲连接数
func WithMaxIdleConns(n int) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.maxIdleConns = n
	}
}

// 设置时区
func WithLoc(loc string) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.loc = url.QueryEscape(loc)
	}
}

// WithDebug 启用调试模式，打印 SQL 语句和参数
func WithDebug() func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.debug = true
	}
}

// debugLog 打印 SQL 语句和参数
func (client *MySQLClient) debugLog(query string, args ...any) {
	if client.debug {
		var argsStr []string
		for _, arg := range args {
			argsStr = append(argsStr, fmt.Sprintf("%v", arg))
		}
		argsJoined := strings.Join(argsStr, ", ")
		zlog.F("sql").Infof("sql:%s, args:[%s]", query, argsJoined)
	}
}

// getFieldsMapping 获取结构体字段映射信息
func (c *MySQLClient) getFieldsMapping(t reflect.Type) map[string]int {
	c.mu.RLock()
	mapping, ok := c.fields[t]
	c.mu.RUnlock()
	if ok {
		return mapping
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	mapping = make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			mapping[dbTag] = i
		}
	}
	c.fields[t] = mapping
	return mapping
}

// scanRows 通用的扫描逻辑
func scanRows(rows *sql.Rows, destValue reflect.Value, sliceElemType reflect.Type) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)
	results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := sliceElemType.Field(fieldIndex).Type
			scanDest[i] = reflect.New(fieldType).Interface()
		} else {
			scanDest[i] = new(any) // 忽略未定义的字段
		}
	}

	for rows.Next() {
		newItem := reflect.New(sliceElemType).Elem()
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		for i, col := range columns {
			if fieldIndex, ok := fieldsMapping[col]; ok {
				field := newItem.Field(fieldIndex)
				value := reflect.ValueOf(scanDest[i]).Elem()
				field.Set(value)
			}
		}
		results = reflect.Append(results, newItem)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %v", err)
	}

	destValue.Elem().Set(results)
	return nil
}

// Find 执行查询并将结果映射到结构体中 列表查询
func Find(dest any, query string, args ...any) error {
	mysql_client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceElemType := destValue.Elem().Type().Elem()
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	return scanRows(rows, destValue, sliceElemType)
}

// FindProc 执行存储过程并将结果映射到结构体中 列表查询
func FindProc(dest any, procName string, args ...any) error {
	mysql_client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceElemType := destValue.Elem().Type().Elem()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	return scanRows(rows, destValue, sliceElemType)
}

// First 执行查询并将结果映射到结构体中，查询一条数据
func First(dest any, query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	structType := destValue.Elem().Type()
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	fieldsMapping := mysql_client.getFieldsMapping(structType)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = reflect.New(fieldType).Interface()
		} else {
			scanDest[i] = new(any)
		}
	}

	if rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("failed to scan row: %v", err)
		}

		for i, col := range columns {
			if fieldIndex, ok := fieldsMapping[col]; ok {
				field := destValue.Elem().Field(fieldIndex)
				value := reflect.ValueOf(scanDest[i]).Elem()
				field.Set(value)
			}
		}
		return true, nil
	}

	return false, nil
}

// FirstProc 执行存储过程并将结果映射到结构体中，查询一条数据
func FirstProc(dest any, procName string, args ...any) (bool, error) {
	mysql_client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	structType := destValue.Elem().Type()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	fieldsMapping := mysql_client.getFieldsMapping(structType)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = reflect.New(fieldType).Interface()
		} else {
			scanDest[i] = new(any)
		}
	}

	if rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("failed to scan row: %v", err)
		}

		for i, col := range columns {
			if fieldIndex, ok := fieldsMapping[col]; ok {
				field := destValue.Elem().Field(fieldIndex)
				value := reflect.ValueOf(scanDest[i]).Elem()
				field.Set(value)
			}
		}
		return true, nil
	}

	return false, nil
}

// FirstCol 执行查询并将单个字段值映射到基础类型
func FirstCol(dest any, query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	if len(columns) != 1 {
		return false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	if rows.Next() {
		var scanDest any
		switch destValue.Elem().Kind() {
		case reflect.String:
			var s string
			scanDest = &s
		case reflect.Int:
			var i int
			scanDest = &i
		case reflect.Int64:
			var i int64
			scanDest = &i
		case reflect.Float64:
			var f float64
			scanDest = &f
		default:
			return false, fmt.Errorf("unsupported type: %s", destValue.Elem().Kind())
		}

		if err := rows.Scan(scanDest); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		destValue.Elem().Set(reflect.ValueOf(scanDest).Elem())
		return true, nil
	}

	return false, nil
}

// FirstColProc 执行存储过程并将单个字段值映射到基础类型
func FirstColProc(dest any, procName string, args ...any) (bool, error) {
	mysql_client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	if len(columns) != 1 {
		return false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	if rows.Next() {
		var scanDest any
		switch destValue.Elem().Kind() {
		case reflect.String:
			var s string
			scanDest = &s
		case reflect.Int:
			var i int
			scanDest = &i
		case reflect.Int64:
			var i int64
			scanDest = &i
		case reflect.Float64:
			var f float64
			scanDest = &f
		default:
			return false, fmt.Errorf("unsupported type: %s", destValue.Elem().Kind())
		}

		if err := rows.Scan(scanDest); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		destValue.Elem().Set(reflect.ValueOf(scanDest).Elem())
		return true, nil
	}

	return false, nil
}

// FindMultipleProc 执行存储过程并将多个结果集映射到多个目标结构体或切片中
func FindMultipleProc(dest []any, procName string, args ...any) error {
	mysql_client.debugLog(procName, args...)

	// 检查目标参数 dest 是否是包含切片或结构体指针的切片
	if len(dest) == 0 {
		return fmt.Errorf("dest cannot be empty")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	for index := 0; ; index++ {
		if index >= len(dest) {
			return fmt.Errorf("too many result sets, expected %d", len(dest))
		}

		destValue := reflect.ValueOf(dest[index])
		if destValue.Kind() != reflect.Ptr {
			return fmt.Errorf("dest[%d] must be a pointer", index)
		}

		kind := destValue.Elem().Kind()
		if kind != reflect.Slice && kind != reflect.Struct {
			return fmt.Errorf("dest[%d] must be a pointer to a struct or slice", index)
		}

		sliceElemType := destValue.Elem().Type()
		if kind == reflect.Slice {
			sliceElemType = sliceElemType.Elem()
		}

		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns: %v", err)
		}

		fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)
		scanDest := make([]any, len(columns))

		for i := range columns {
			if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
				fieldType := sliceElemType.Field(fieldIndex).Type
				scanDest[i] = reflect.New(fieldType).Interface()
			} else {
				scanDest[i] = new(any) // 忽略未定义的字段
			}
		}

		if kind == reflect.Slice {
			results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)

			for rows.Next() {
				newItem := reflect.New(sliceElemType).Elem()
				if err := rows.Scan(scanDest...); err != nil {
					return fmt.Errorf("failed to scan row: %v", err)
				}

				for i, col := range columns {
					if fieldIndex, ok := fieldsMapping[col]; ok {
						field := newItem.Field(fieldIndex)
						value := reflect.ValueOf(scanDest[i]).Elem()
						field.Set(value)
					}
				}

				results = reflect.Append(results, newItem)
			}

			destValue.Elem().Set(results)
		} else {
			if rows.Next() {
				newItem := reflect.New(sliceElemType).Elem()
				if err := rows.Scan(scanDest...); err != nil {
					return fmt.Errorf("failed to scan row: %v", err)
				}

				for i, col := range columns {
					if fieldIndex, ok := fieldsMapping[col]; ok {
						field := newItem.Field(fieldIndex)
						value := reflect.ValueOf(scanDest[i]).Elem()
						field.Set(value)
					}
				}

				destValue.Elem().Set(newItem)
			}
		}

		if !rows.NextResultSet() {
			break
		}
	}

	return nil
}

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

// Close 关闭数据库连接
func Close() error {
	return mysql_client.DB.Close()
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
