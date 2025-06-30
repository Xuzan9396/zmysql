package smysql

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

type IS_LIST_TYPE int8

const (
	HAS_ONE IS_LIST_TYPE = iota + 1
	HAS_LIST
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

// Conn 创建并初始化一个新的 MySQL 客户端
func Conn(username, password, addr, dbName string, opts ...func(*MySQLClient)) (*MySQLClient, error) {
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
		return nil, fmt.Errorf("failed to open database: %v", err)
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

	return client, nil
}

// WithConnMaxLifetime 设置连接最大生命周期
func WithConnMaxLifetime(d time.Duration) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.connMaxLifetime = d
	}
}

// WithMaxOpenConns 设置最大连接数
func WithMaxOpenConns(n int) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.maxOpenConns = n
	}
}

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(n int) func(*MySQLClient) {
	return func(client *MySQLClient) {
		client.maxIdleConns = n
	}
}

// WithLoc 设置时区
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

// createNullScanner 根据字段类型创建对应的Null扫描器，只处理基本类型
func (client *MySQLClient) createNullScanner(fieldType reflect.Type) any {
	switch fieldType.Kind() {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &sql.NullInt64{}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &sql.NullInt64{}
	case reflect.Float32, reflect.Float64:
		return &sql.NullFloat64{}
	case reflect.Bool:
		return &sql.NullBool{}
	default:
		return reflect.New(fieldType).Interface()
	}
}

// setFieldFromNullScanner 从Null扫描器设置字段值
func (client *MySQLClient) setFieldFromNullScanner(field reflect.Value, scanner any, fieldType reflect.Type) error {
	switch s := scanner.(type) {
	case *sql.NullString:
		if s.Valid {
			if fieldType.Kind() != reflect.String {
				return fmt.Errorf("cannot convert string to %s", fieldType.Kind())
			}
			field.SetString(s.String)
		} else {
			field.SetString("")
		}
	case *sql.NullInt64:
		if s.Valid {
			switch fieldType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				field.SetInt(s.Int64)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if s.Int64 < 0 {
					return fmt.Errorf("cannot convert negative value to unsigned type")
				}
				field.SetUint(uint64(s.Int64))
			default:
				return fmt.Errorf("cannot convert int64 to %s", fieldType.Kind())
			}
		} else {
			switch fieldType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				field.SetInt(0)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				field.SetUint(0)
			}
		}
	case *sql.NullFloat64:
		if s.Valid {
			if fieldType.Kind() != reflect.Float32 && fieldType.Kind() != reflect.Float64 {
				return fmt.Errorf("cannot convert float64 to %s", fieldType.Kind())
			}
			field.SetFloat(s.Float64)
		} else {
			field.SetFloat(0.0)
		}
	case *sql.NullBool:
		if s.Valid {
			if fieldType.Kind() != reflect.Bool {
				return fmt.Errorf("cannot convert bool to %s", fieldType.Kind())
			}
			field.SetBool(s.Bool)
		} else {
			field.SetBool(false)
		}
	default:
		value := reflect.ValueOf(scanner).Elem()
		field.Set(value)
	}
	return nil
}

// scanRows 通用的扫描逻辑
func (client *MySQLClient) scanRows(rows *sql.Rows, destValue reflect.Value, sliceElemType reflect.Type) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	fieldsMapping := client.getFieldsMapping(sliceElemType)
	results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := sliceElemType.Field(fieldIndex).Type
			scanDest[i] = client.createNullScanner(fieldType)
		} else {
			scanDest[i] = &sql.NullString{}
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
				fieldType := sliceElemType.Field(fieldIndex).Type
				if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
					return fmt.Errorf("failed to set field %s: %v", col, err)
				}
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
func (client *MySQLClient) Find(dest any, query string, args ...any) error {
	client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceElemType := destValue.Elem().Type().Elem()
	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	return client.scanRows(rows, destValue, sliceElemType)
}

// FindProc 执行存储过程并将结果映射到结构体中 列表查询
func (client *MySQLClient) FindProc(dest any, procName string, args ...any) error {
	client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceElemType := destValue.Elem().Type().Elem()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	return client.scanRows(rows, destValue, sliceElemType)
}

// First 执行查询并将结果映射到结构体中，查询一条数据
func (client *MySQLClient) First(dest any, query string, args ...any) (bool, error) {
	client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	structType := destValue.Elem().Type()
	stmt, err := client.DB.Prepare(query)
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

	fieldsMapping := client.getFieldsMapping(structType)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = client.createNullScanner(fieldType)
		} else {
			scanDest[i] = &sql.NullString{}
		}
	}

	if rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("failed to scan row: %v", err)
		}

		for i, col := range columns {
			if fieldIndex, ok := fieldsMapping[col]; ok {
				field := destValue.Elem().Field(fieldIndex)
				fieldType := structType.Field(fieldIndex).Type
				if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
					return false, fmt.Errorf("failed to set field %s: %v", col, err)
				}
			}
		}
		return true, nil
	}

	return false, nil
}

// FirstProc 执行存储过程并将结果映射到结构体中，查询一条数据
func (client *MySQLClient) FirstProc(dest any, procName string, args ...any) (bool, error) {
	client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	structType := destValue.Elem().Type()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
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

	fieldsMapping := client.getFieldsMapping(structType)
	scanDest := make([]any, len(columns))

	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = client.createNullScanner(fieldType)
		} else {
			scanDest[i] = &sql.NullString{}
		}
	}

	if rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("failed to scan row: %v", err)
		}

		for i, col := range columns {
			if fieldIndex, ok := fieldsMapping[col]; ok {
				field := destValue.Elem().Field(fieldIndex)
				fieldType := structType.Field(fieldIndex).Type
				if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
					return false, fmt.Errorf("failed to set field %s: %v", col, err)
				}
			}
		}
		return true, nil
	}

	return false, nil
}

// FirstCol 执行查询并将单个字段值映射到基础类型
func (client *MySQLClient) FirstCol(dest any, query string, args ...any) (bool, error) {
	client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}

	stmt, err := client.DB.Prepare(query)
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
		scanner := client.createNullScanner(destValue.Elem().Type())
		if err := rows.Scan(scanner); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		if err := client.setFieldFromNullScanner(destValue.Elem(), scanner, destValue.Elem().Type()); err != nil {
			return false, fmt.Errorf("failed to set value: %v", err)
		}
		return true, nil
	}

	return false, nil
}

// FirstColProc 执行存储过程并将单个字段值映射到基础类型
func (client *MySQLClient) FirstColProc(dest any, procName string, args ...any) (bool, error) {
	client.debugLog(procName, args...)
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
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
		scanner := client.createNullScanner(destValue.Elem().Type())
		if err := rows.Scan(scanner); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		if err := client.setFieldFromNullScanner(destValue.Elem(), scanner, destValue.Elem().Type()); err != nil {
			return false, fmt.Errorf("failed to set value: %v", err)
		}
		return true, nil
	}

	return false, nil
}

// Exec 执行查询并返回是否成功
func (client *MySQLClient) Exec(query string, args ...any) (bool, error) {
	client.debugLog(query, args...)
	stmt, err := client.DB.Prepare(query)
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
func (client *MySQLClient) ExecByte(query string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	client.debugLog(query, args...)
	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	var resultData []map[string]any

	for rows.Next() {
		rowData := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowData {
			rowPointers[i] = &rowData[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			rowMap[col] = rowData[i]
		}

		resultData = append(resultData, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if isList == HAS_ONE && len(resultData) >= 1 {
		jsonData, err := json.Marshal(resultData[0])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal single result data: %v", err)
		}
		return jsonData, nil
	}

	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result data: %v", err)
	}

	return jsonData, nil
}

// ExecProcByte 执行存储过程并返回结果集的所有数据，格式为 []byte
func (client *MySQLClient) ExecProcByte(procName string, isList IS_LIST_TYPE, args ...any) ([]byte, error) {
	client.debugLog(procName, args...)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	var resultData []map[string]any

	for rows.Next() {
		rowData := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowData {
			rowPointers[i] = &rowData[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			rowMap[col] = rowData[i]
		}

		resultData = append(resultData, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if isList == HAS_ONE && len(resultData) >= 1 {
		jsonData, err := json.Marshal(resultData[0])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal single result data: %v", err)
		}
		return jsonData, nil
	}

	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result data: %v", err)
	}

	return jsonData, nil
}

// FindMultipleProc 执行存储过程并将多个结果集映射到多个目标结构体或切片中
func (client *MySQLClient) FindMultipleProc(dest []any, procName string, args ...any) error {
	client.debugLog(procName, args...)

	if len(dest) == 0 {
		return fmt.Errorf("dest cannot be empty")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
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

		if kind == reflect.Slice {
			if err := client.scanRows(rows, destValue, sliceElemType); err != nil {
				return fmt.Errorf("failed to scan result set %d: %v", index, err)
			}
		} else {
			columns, err := rows.Columns()
			if err != nil {
				return fmt.Errorf("failed to get columns: %v", err)
			}

			fieldsMapping := client.getFieldsMapping(sliceElemType)
			scanDest := make([]any, len(columns))

			for i := range columns {
				if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
					fieldType := sliceElemType.Field(fieldIndex).Type
					scanDest[i] = client.createNullScanner(fieldType)
				} else {
					scanDest[i] = &sql.NullString{}
				}
			}

			if rows.Next() {
				if err := rows.Scan(scanDest...); err != nil {
					return fmt.Errorf("failed to scan row: %v", err)
				}

				newItem := reflect.New(sliceElemType).Elem()
				for i, col := range columns {
					if fieldIndex, ok := fieldsMapping[col]; ok {
						field := newItem.Field(fieldIndex)
						fieldType := sliceElemType.Field(fieldIndex).Type
						if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
							return fmt.Errorf("failed to set field %s: %v", col, err)
						}
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

// ExecFindLastId 执行SQL查询并返回LastInsertId
func (client *MySQLClient) ExecFindLastId(query string, args ...any) (int64, error) {
	client.debugLog(query, args...)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %v", err)
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %v", err)
	}

	return lastId, nil
}

// FirstColProcInt64 执行存储过程并将单个字段值映射到int64类型
func (client *MySQLClient) FirstColProcInt64(procName string, args ...any) (int64, bool, error) {
	return firstColProcAny[int64](client, procName, args...)
}

// FirstColProcString 执行存储过程并将单个字段值映射到string类型
func (client *MySQLClient) FirstColProcString(procName string, args ...any) (string, bool, error) {
	return firstColProcAny[string](client, procName, args...)
}

// FirstColInt64 执行查询并将单个字段值映射到int64类型
func (client *MySQLClient) FirstColInt64(query string, args ...any) (int64, bool, error) {
	return firstColAny[int64](client, query, args...)
}

// FirstColString 执行查询并将单个字段值映射到string类型
func (client *MySQLClient) FirstColString(query string, args ...any) (string, bool, error) {
	return firstColAny[string](client, query, args...)
}

// firstColAny 执行查询并将单个字段值映射到泛型类型 - 包级泛型函数
func firstColAny[T int64 | string](client *MySQLClient, query string, args ...any) (T, bool, error) {
	client.debugLog(query, args...)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to get columns: %v", err)
	}

	if len(columns) != 1 {
		var zero T
		return zero, false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	if rows.Next() {
		var t T
		switch any(t).(type) {
		case int64:
			var nullInt sql.NullInt64
			if err := rows.Scan(&nullInt); err != nil {
				var zero T
				return zero, false, fmt.Errorf("failed to scan column: %v", err)
			}
			if nullInt.Valid {
				return any(nullInt.Int64).(T), true, nil
			} else {
				var zero T
				return zero, true, nil
			}
		case string:
			var nullStr sql.NullString
			if err := rows.Scan(&nullStr); err != nil {
				var zero T
				return zero, false, fmt.Errorf("failed to scan column: %v", err)
			}
			if nullStr.Valid {
				return any(nullStr.String).(T), true, nil
			} else {
				var zero T
				return zero, true, nil
			}
		}
	}

	var zero T
	return zero, false, nil
}


// firstColProcAny 执行存储过程并将单个字段值映射到泛型类型 - 包级泛型函数
func firstColProcAny[T int64 | string](client *MySQLClient, procName string, args ...any) (T, bool, error) {
	client.debugLog(procName, args...)

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		var zero T
		return zero, false, fmt.Errorf("failed to get columns: %v", err)
	}

	if len(columns) != 1 {
		var zero T
		return zero, false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	if rows.Next() {
		var t T
		switch any(t).(type) {
		case int64:
			var nullInt sql.NullInt64
			if err := rows.Scan(&nullInt); err != nil {
				var zero T
				return zero, false, fmt.Errorf("failed to scan column: %v", err)
			}
			if nullInt.Valid {
				return any(nullInt.Int64).(T), true, nil
			} else {
				var zero T
				return zero, true, nil
			}
		case string:
			var nullStr sql.NullString
			if err := rows.Scan(&nullStr); err != nil {
				var zero T
				return zero, false, fmt.Errorf("failed to scan column: %v", err)
			}
			if nullStr.Valid {
				return any(nullStr.String).(T), true, nil
			} else {
				var zero T
				return zero, true, nil
			}
		}
	}

	var zero T
	return zero, false, nil
}


// findArray 执行查询并返回指定字段的泛型数组 - 包级泛型函数
func findArray[T int64 | string](client *MySQLClient, fieldName string, query string, args ...any) ([]T, error) {
	client.debugLog(query, args...)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 查找字段索引
	fieldIndex := -1
	for i, col := range columns {
		if col == fieldName {
			fieldIndex = i
			break
		}
	}

	if fieldIndex == -1 {
		return nil, fmt.Errorf("field '%s' not found in query results", fieldName)
	}

	var results []T
	scanDest := make([]any, len(columns))

	for rows.Next() {
		// 为每列创建扫描目标
		for i := range columns {
			if i == fieldIndex {
				var t T
				switch any(t).(type) {
				case int64:
					scanDest[i] = &sql.NullInt64{}
				case string:
					scanDest[i] = &sql.NullString{}
				}
			} else {
				scanDest[i] = &sql.NullString{} // 忽略其他字段
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 获取目标字段值
		var t T
		switch any(t).(type) {
		case int64:
			nullInt := scanDest[fieldIndex].(*sql.NullInt64)
			if nullInt.Valid {
				results = append(results, any(nullInt.Int64).(T))
			}
		case string:
			nullStr := scanDest[fieldIndex].(*sql.NullString)
			if nullStr.Valid {
				results = append(results, any(nullStr.String).(T))
			}
		}
		// NULL值不添加到结果切片中
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results, nil
}


// findProcArray 执行存储过程并返回指定字段的泛型数组 - 包级泛型函数
func findProcArray[T int64 | string](client *MySQLClient, fieldName string, procName string, args ...any) ([]T, error) {
	client.debugLog(procName, args...)

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 查找字段索引
	fieldIndex := -1
	for i, col := range columns {
		if col == fieldName {
			fieldIndex = i
			break
		}
	}

	if fieldIndex == -1 {
		return nil, fmt.Errorf("field '%s' not found in query results", fieldName)
	}

	var results []T
	scanDest := make([]any, len(columns))

	for rows.Next() {
		// 为每列创建扫描目标
		for i := range columns {
			if i == fieldIndex {
				var t T
				switch any(t).(type) {
				case int64:
					scanDest[i] = &sql.NullInt64{}
				case string:
					scanDest[i] = &sql.NullString{}
				}
			} else {
				scanDest[i] = &sql.NullString{} // 忽略其他字段
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 获取目标字段值
		var t T
		switch any(t).(type) {
		case int64:
			nullInt := scanDest[fieldIndex].(*sql.NullInt64)
			if nullInt.Valid {
				results = append(results, any(nullInt.Int64).(T))
			}
		case string:
			nullStr := scanDest[fieldIndex].(*sql.NullString)
			if nullStr.Valid {
				results = append(results, any(nullStr.String).(T))
			}
		}
		// NULL值不添加到结果切片中
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results, nil
}


// FindArrayInt64 执行查询并返回指定字段的int64数组
func (client *MySQLClient) FindArrayInt64(fieldName string, query string, args ...any) ([]int64, error) {
	return findArray[int64](client, fieldName, query, args...)
}

// FindArrayString 执行查询并返回指定字段的string数组
func (client *MySQLClient) FindArrayString(fieldName string, query string, args ...any) ([]string, error) {
	return findArray[string](client, fieldName, query, args...)
}

// FindProcArrayInt64 执行存储过程并返回指定字段的int64数组
func (client *MySQLClient) FindProcArrayInt64(fieldName string, procName string, args ...any) ([]int64, error) {
	return findProcArray[int64](client, fieldName, procName, args...)
}

// FindProcArrayString 执行存储过程并返回指定字段的string数组
func (client *MySQLClient) FindProcArrayString(fieldName string, procName string, args ...any) ([]string, error) {
	return findProcArray[string](client, fieldName, procName, args...)
}

// findMap 执行查询并返回map[T]Y，支持泛型键值类型 - 包级泛型函数
// keyField: 键字段名，不能为空
// valueField: 值字段名，为空时返回整个结构体
// key为null的行会被过滤掉
func findMap[T comparable, Y any](client *MySQLClient, keyField string, valueField string, query string, args ...any) (map[T]Y, error) {
	client.debugLog(query, args...)

	if keyField == "" {
		return nil, fmt.Errorf("keyField cannot be empty")
	}

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 查找键字段索引
	keyIndex := -1
	valueIndex := -1
	for i, col := range columns {
		if col == keyField {
			keyIndex = i
		}
		if valueField != "" && col == valueField {
			valueIndex = i
		}
	}

	if keyIndex == -1 {
		return nil, fmt.Errorf("key field '%s' not found in query results", keyField)
	}
	if valueField != "" && valueIndex == -1 {
		return nil, fmt.Errorf("value field '%s' not found in query results", valueField)
	}

	result := make(map[T]Y)
	scanDest := make([]any, len(columns))

	// 检查Y是否为结构体类型
	var yType reflect.Type
	var yValue Y
	yValueRef := reflect.ValueOf(&yValue).Elem()
	yType = yValueRef.Type()
	isStructType := yType.Kind() == reflect.Struct

	// 如果Y是结构体类型，获取字段映射
	var fieldsMapping map[string]int
	if isStructType {
		fieldsMapping = client.getFieldsMapping(yType)
	}

	for rows.Next() {
		// 为每列创建扫描目标
		for i := range columns {
			if isStructType && valueField == "" {
				// 结构体模式，根据字段类型创建扫描器
				if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
					fieldType := yType.Field(fieldIndex).Type
					scanDest[i] = client.createNullScanner(fieldType)
				} else {
					scanDest[i] = &sql.NullString{}
				}
			} else {
				// 基础类型模式，使用NullString扫描器
				scanDest[i] = &sql.NullString{}
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 获取键值，如果键为null则跳过该行
		var key T
		keyScanner := scanDest[keyIndex]
		keyValid := false

		switch s := keyScanner.(type) {
		case *sql.NullString:
			if s.Valid {
				keyValid = true
				// 尝试转换为T类型
				switch any(key).(type) {
				case string:
					key = any(s.String).(T)
				case int64:
					// 需要转换字符串为int64
					var val int64
					if _, err := fmt.Sscanf(s.String, "%d", &val); err == nil {
						key = any(val).(T)
					}
				}
			}
		case *sql.NullInt64:
			if s.Valid {
				keyValid = true
				switch any(key).(type) {
				case int64:
					key = any(s.Int64).(T)
				case string:
					key = any(fmt.Sprintf("%d", s.Int64)).(T)
				}
			}
		}

		// 如果键为null，跳过该行
		if !keyValid {
			continue
		}

		// 构建值
		var value Y
		if valueField == "" && isStructType {
			// 返回整个结构体
			newStruct := reflect.New(yType).Elem()
			for i, col := range columns {
				if fieldIndex, ok := fieldsMapping[col]; ok {
					field := newStruct.Field(fieldIndex)
					fieldType := yType.Field(fieldIndex).Type
					if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
						return nil, fmt.Errorf("failed to set field %s: %v", col, err)
					}
				}
			}
			value = newStruct.Interface().(Y)
		} else {
			// 返回指定字段值
			valueScanner := scanDest[valueIndex].(*sql.NullString)
			if valueScanner.Valid {
				// 尝试转换为Y类型
				switch any(value).(type) {
				case string:
					value = any(valueScanner.String).(Y)
				case int64:
					var val int64
					if _, err := fmt.Sscanf(valueScanner.String, "%d", &val); err == nil {
						value = any(val).(Y)
					}
				}
			} else {
				// 值为null，使用Y类型的零值
				var zero Y
				value = zero
			}
		}

		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}


// findProcMap 执行存储过程并返回map[T]Y，支持泛型键值类型 - 包级泛型函数
// keyField: 键字段名，不能为空
// valueField: 值字段名，为空时返回整个结构体
// key为null的行会被过滤掉
func findProcMap[T comparable, Y any](client *MySQLClient, keyField string, valueField string, procName string, args ...any) (map[T]Y, error) {
	client.debugLog(procName, args...)

	if keyField == "" {
		return nil, fmt.Errorf("keyField cannot be empty")
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)

	stmt, err := client.DB.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// 查找键字段索引
	keyIndex := -1
	valueIndex := -1
	for i, col := range columns {
		if col == keyField {
			keyIndex = i
		}
		if valueField != "" && col == valueField {
			valueIndex = i
		}
	}

	if keyIndex == -1 {
		return nil, fmt.Errorf("key field '%s' not found in query results", keyField)
	}
	if valueField != "" && valueIndex == -1 {
		return nil, fmt.Errorf("value field '%s' not found in query results", valueField)
	}

	result := make(map[T]Y)
	scanDest := make([]any, len(columns))

	// 检查Y是否为结构体类型
	var yType reflect.Type
	var yValue Y
	yValueRef := reflect.ValueOf(&yValue).Elem()
	yType = yValueRef.Type()
	isStructType := yType.Kind() == reflect.Struct

	// 如果Y是结构体类型，获取字段映射
	var fieldsMapping map[string]int
	if isStructType {
		fieldsMapping = client.getFieldsMapping(yType)
	}

	for rows.Next() {
		// 为每列创建扫描目标
		for i := range columns {
			if isStructType && valueField == "" {
				// 结构体模式，根据字段类型创建扫描器
				if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
					fieldType := yType.Field(fieldIndex).Type
					scanDest[i] = client.createNullScanner(fieldType)
				} else {
					scanDest[i] = &sql.NullString{}
				}
			} else {
				// 基础类型模式，使用NullString扫描器
				scanDest[i] = &sql.NullString{}
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// 获取键值，如果键为null则跳过该行
		var key T
		keyScanner := scanDest[keyIndex]
		keyValid := false

		switch s := keyScanner.(type) {
		case *sql.NullString:
			if s.Valid {
				keyValid = true
				// 尝试转换为T类型
				switch any(key).(type) {
				case string:
					key = any(s.String).(T)
				case int64:
					// 需要转换字符串为int64
					var val int64
					if _, err := fmt.Sscanf(s.String, "%d", &val); err == nil {
						key = any(val).(T)
					}
				}
			}
		case *sql.NullInt64:
			if s.Valid {
				keyValid = true
				switch any(key).(type) {
				case int64:
					key = any(s.Int64).(T)
				case string:
					key = any(fmt.Sprintf("%d", s.Int64)).(T)
				}
			}
		}

		// 如果键为null，跳过该行
		if !keyValid {
			continue
		}

		// 构建值
		var value Y
		if valueField == "" && isStructType {
			// 返回整个结构体
			newStruct := reflect.New(yType).Elem()
			for i, col := range columns {
				if fieldIndex, ok := fieldsMapping[col]; ok {
					field := newStruct.Field(fieldIndex)
					fieldType := yType.Field(fieldIndex).Type
					if err := client.setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
						return nil, fmt.Errorf("failed to set field %s: %v", col, err)
					}
				}
			}
			value = newStruct.Interface().(Y)
		} else {
			// 返回指定字段值
			valueScanner := scanDest[valueIndex].(*sql.NullString)
			if valueScanner.Valid {
				// 尝试转换为Y类型
				switch any(value).(type) {
				case string:
					value = any(valueScanner.String).(Y)
				case int64:
					var val int64
					if _, err := fmt.Sscanf(valueScanner.String, "%d", &val); err == nil {
						value = any(val).(Y)
					}
				}
			} else {
				// 值为null，使用Y类型的零值
				var zero Y
				value = zero
			}
		}

		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

// 包级泛型函数，由于 Go 不支持方法泛型

// FindArray 执行查询并返回指定字段的泛型数组 - 包级函数
func FindArray[T int64 | string](client *MySQLClient, fieldName string, query string, args ...any) ([]T, error) {
	return findArray[T](client, fieldName, query, args...)
}

// FindProcArray 执行存储过程并返回指定字段的泛型数组 - 包级函数
func FindProcArray[T int64 | string](client *MySQLClient, fieldName string, procName string, args ...any) ([]T, error) {
	return findProcArray[T](client, fieldName, procName, args...)
}

// FirstColAny 执行查询并将单个字段值映射到指定类型（泛型版本）- 包级函数
func FirstColAny[T int64 | string](client *MySQLClient, query string, args ...any) (T, bool, error) {
	return firstColAny[T](client, query, args...)
}

// FirstColProcAny 执行存储过程并将单个字段值映射到指定类型（泛型版本）- 包级函数
func FirstColProcAny[T int64 | string](client *MySQLClient, procName string, args ...any) (T, bool, error) {
	return firstColProcAny[T](client, procName, args...)
}

// FindMap 执行查询并返回map[T]Y，支持泛型键值类型 - 包级函数
func FindMap[T comparable, Y any](client *MySQLClient, keyField string, valueField string, query string, args ...any) (map[T]Y, error) {
	return findMap[T, Y](client, keyField, valueField, query, args...)
}

// FindProcMap 执行存储过程并返回map[T]Y，支持泛型键值类型 - 包级函数
func FindProcMap[T comparable, Y any](client *MySQLClient, keyField string, valueField string, procName string, args ...any) (map[T]Y, error) {
	return findProcMap[T, Y](client, keyField, valueField, procName, args...)
}

// Close 关闭数据库连接
func (client *MySQLClient) Close() error {
	return client.DB.Close()
}