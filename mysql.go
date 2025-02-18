package zmysql

import (
	"database/sql"
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

// 创建并初始化一个新的 MySQL 客户端
func Conn(username, password, addr, dbName string, opts ...func(*MySQLClient)) error {

	// 创建默认的 MySQL 客户端
	client := &MySQLClient{
		//DB:              db,
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

	// 返回客户端实例
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

		// 将args中的每个元素转换为字符串，并用逗号分隔
		var argsStr []string
		for _, arg := range args {
			argsStr = append(argsStr, fmt.Sprintf("%v", arg))
		}
		// 将argsStr中的元素用逗号连接成一个字符串
		argsJoined := strings.Join(argsStr, ", ")

		// 打印 SQL 语句和格式化后的参数
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

// Find 执行查询并将结果映射到结构体中 列表查询
func Find(dest any, query string, args ...any) error {
	// 检查目标参数 dest 是否是指向切片的指针
	mysql_client.debugLog(query, args...)
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向切片的指针
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	// 获取目标切片元素类型（sliceElemType 即结构体类型）
	sliceElemType := destValue.Elem().Type().Elem()

	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns() // 获取字段名称
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	// 获取字段映射缓存（假设有一个缓存用于存储列名与结构体字段索引的映射）
	fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)

	// 预分配内存，创建一个空的目标切片 创建一个类型为 []CityInfo，但长度和容量都为 0 的切片
	results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)

	// 创建扫描目标数组，用于存放每一列的值
	scanDest := make([]any, len(columns))
	for i := range columns {
		// 判断字段是否存在映射  fieldsMapping[columns[i]]:为结构体字段索引
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			// 获取结构体字段类型
			fieldType := sliceElemType.Field(fieldIndex).Type
			scanDest[i] = reflect.New(fieldType).Interface()
		}
	}

	// 遍历每一行结果
	for rows.Next() {
		// 创建新的结构体实例  reflect.New(sliceElemType) 会返回一个 *CityInfo 类型的指针
		newItem := reflect.New(sliceElemType).Elem()

		// 执行扫描，将当前行数据扫描到 scanDest 数组中
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("结构体和查询字段不对应: %v", err)
		}

		// 将扫描到的数据填充到结构体字段
		for i, col := range columns {
			// 判断列名是否有对应的字段映射
			if fieldIndex, ok := fieldsMapping[col]; ok {
				// 获取结构体中对应字段
				field := newItem.Field(fieldIndex)
				// 获取扫描到的值
				value := reflect.ValueOf(scanDest[i]).Elem()

				field.Set(value)
			}
		}

		// 将当前结构体添加到目标切片中
		results = reflect.Append(results, newItem)
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %v", err)
	}
	// 设置目标切片为查询结果 destValue.Elem() 返回的是 []CityInfo
	destValue.Elem().Set(results)
	return nil
}

func FindProc(dest any, procName string, args ...any) error {
	mysql_client.debugLog(procName, args...)

	// 检查目标参数 dest 是否是指向切片的指针
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向切片的指针
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	// 获取目标切片元素类型（sliceElemType 即结构体类型）
	sliceElemType := destValue.Elem().Type().Elem()

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders) // 使用预处理来避免重复解析 SQL 查询

	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns() // 获取字段名称
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	// 获取字段映射缓存（假设有一个缓存用于存储列名与结构体字段索引的映射）
	fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)

	// 预分配内存，创建一个空的目标切片 创建一个类型为 []CityInfo，但长度和容量都为 0 的切片
	results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)

	// 创建扫描目标数组，用于存放每一列的值
	scanDest := make([]any, len(columns))
	for i := range columns {
		// 判断字段是否存在映射  fieldsMapping[columns[i]]:为结构体字段索引
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			// 获取结构体字段类型
			fieldType := sliceElemType.Field(fieldIndex).Type
			scanDest[i] = reflect.New(fieldType).Interface()
		}
	}

	// 遍历每一行结果
	for rows.Next() {
		// 创建新的结构体实例  reflect.New(sliceElemType) 会返回一个 *CityInfo 类型的指针
		newItem := reflect.New(sliceElemType).Elem()

		// 执行扫描，将当前行数据扫描到 scanDest 数组中
		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("结构体和查询字段不对应: %v", err)
		}

		// 将扫描到的数据填充到结构体字段
		for i, col := range columns {
			// 判断列名是否有对应的字段映射
			if fieldIndex, ok := fieldsMapping[col]; ok {
				// 获取结构体中对应字段
				field := newItem.Field(fieldIndex)
				// 获取扫描到的值
				value := reflect.ValueOf(scanDest[i]).Elem()

				field.Set(value)
			}
		}

		// 将当前结构体添加到目标切片中
		results = reflect.Append(results, newItem)
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %v", err)
	}
	// 设置目标切片为查询结果 destValue.Elem() 返回的是 []CityInfo
	destValue.Elem().Set(results)
	return nil
}

// 执行查询并将结果映射到结构体中，查询一条数据
func First(dest any, query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)

	// 检查目标参数 dest 是否是指向结构体的指针
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向结构体的指针
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	// 获取目标结构体类型
	structType := destValue.Elem().Type()

	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	// 获取字段映射缓存（假设有一个缓存用于存储列名与结构体字段索引的映射）
	fieldsMapping := mysql_client.getFieldsMapping(structType)

	// 创建扫描目标数组，用于存放每一列的值
	scanDest := make([]any, len(columns))
	for i := range columns {
		// 判断字段是否存在映射
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			// 获取结构体字段类型
			fieldType := structType.Field(fieldIndex).Type

			scanDest[i] = reflect.New(fieldType).Interface()
		}
	}

	// 只扫描第一行数据
	if rows.Next() {
		// 执行扫描，将当前行数据扫描到 scanDest 数组中
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("structure and query fields do not match: %v", err)
		}

		// 将扫描到的数据填充到结构体字段
		for i, col := range columns {
			// 判断列名是否有对应的字段映射
			if fieldIndex, ok := fieldsMapping[col]; ok {
				// 获取结构体中对应字段
				field := reflect.ValueOf(dest).Elem().Field(fieldIndex)
				// 获取扫描到的值
				value := reflect.ValueOf(scanDest[i]).Elem()
				field.Set(value)
			}
		}
	} else {
		// 如果没有查询到任何记录 ,为false
		return false, nil
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("rows iteration error: %v", err)
	}

	return true, nil
}

// 执行查询并将结果映射到结构体中，查询一条数据
func FirstProc(dest any, procName string, args ...any) (bool, error) {
	mysql_client.debugLog(procName, args...)

	// 检查目标参数 dest 是否是指向结构体的指针
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向结构体的指针
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("dest must be a pointer to a struct")
	}

	// 获取目标结构体类型
	structType := destValue.Elem().Type()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)
	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	// 获取字段映射缓存（假设有一个缓存用于存储列名与结构体字段索引的映射）
	fieldsMapping := mysql_client.getFieldsMapping(structType)

	// 创建扫描目标数组，用于存放每一列的值
	scanDest := make([]any, len(columns))
	for i := range columns {
		// 判断字段是否存在映射
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			// 获取结构体字段类型
			fieldType := structType.Field(fieldIndex).Type

			scanDest[i] = reflect.New(fieldType).Interface()
		}
	}

	// 只扫描第一行数据
	if rows.Next() {
		// 执行扫描，将当前行数据扫描到 scanDest 数组中
		if err := rows.Scan(scanDest...); err != nil {
			return false, fmt.Errorf("structure and query fields do not match: %v", err)
		}

		// 将扫描到的数据填充到结构体字段
		for i, col := range columns {
			// 判断列名是否有对应的字段映射
			if fieldIndex, ok := fieldsMapping[col]; ok {
				// 获取结构体中对应字段
				field := reflect.ValueOf(dest).Elem().Field(fieldIndex)
				// 获取扫描到的值
				value := reflect.ValueOf(scanDest[i]).Elem()
				field.Set(value)
			}
		}
	} else {
		// 如果没有查询到任何记录 ,为false
		return false, nil
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("rows iteration error: %v", err)
	}

	return true, nil
}

// 执行查询并将单个字段值映射到结构体的字段
func FirstCol(dest any, query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)

	// 检查目标参数 dest 是否是指向基础类型的指针
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向基础类型的指针
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}

	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	// 只查询一个字段的值
	if len(columns) != 1 {
		return false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	// 获取查询到的值并扫描
	if rows.Next() {
		// 创建一个与目标类型匹配的指针
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
		case reflect.Int8:
			var i int8
			scanDest = &i
		case reflect.Uint8:
			var i uint8
			scanDest = &i
		case reflect.Float64:
			var f float64
			scanDest = &f

		default:
			return false, fmt.Errorf("unsupported type: %s", destValue.Elem().Kind())
		}

		// 执行扫描，将当前行数据扫描到 scanDest 中
		if err = rows.Scan(scanDest); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		// 将扫描到的值赋值到基础类型指针
		destValue.Elem().Set(reflect.ValueOf(scanDest).Elem())

	} else {
		// 如果没有查询到任何数据
		return false, nil
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("rows iteration error: %v", err)
	}

	return true, nil
}

// 执行查询并将单个字段值映射到结构体的字段
func FirstColProc(dest any, procName string, args ...any) (bool, error) {
	mysql_client.debugLog(procName, args...)

	// 检查目标参数 dest 是否是指向基础类型的指针
	destValue := reflect.ValueOf(dest)
	// 目标类型需要是指向基础类型的指针
	if destValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dest must be a pointer to a basic type")
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)
	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 获取查询结果的列名
	columns, err := rows.Columns()
	if err != nil {
		return false, fmt.Errorf("failed to get columns: %v", err)
	}

	// 只查询一个字段的值
	if len(columns) != 1 {
		return false, fmt.Errorf("expected a single column result, but got %d columns", len(columns))
	}

	// 获取查询到的值并扫描
	if rows.Next() {
		// 创建一个与目标类型匹配的指针
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
		case reflect.Int8:
			var i int8
			scanDest = &i
		case reflect.Uint8:
			var i uint8
			scanDest = &i
		case reflect.Float64:
			var f float64
			scanDest = &f

		default:
			return false, fmt.Errorf("unsupported type: %s", destValue.Elem().Kind())
		}

		// 执行扫描，将当前行数据扫描到 scanDest 中
		if err = rows.Scan(scanDest); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		// 将扫描到的值赋值到基础类型指针
		destValue.Elem().Set(reflect.ValueOf(scanDest).Elem())

	} else {
		// 如果没有查询到任何数据
		return false, nil
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("rows iteration error: %v", err)
	}

	return true, nil
}

func FindMultipleProc(dest []any, procName string, args ...any) error {
	mysql_client.debugLog(procName, args...)

	// 检查目标参数 dest 是否是包含切片或结构体指针的切片
	if len(dest) == 0 {
		return fmt.Errorf("dest cannot be empty")
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	query := fmt.Sprintf("CALL `%s`(%s)", procName, placeholders)
	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(args...)
	stmt.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// 遍历每个结果集
	for index := 0; ; index++ {
		// 如果目标切片已经处理完所有结果集，则退出循环
		if index >= len(dest) {
			// 如果结果集数量超过了传入的切片数量，返回错误
			return fmt.Errorf("too many result sets, expected %d", len(dest))
		}

		// 获取目标结构体类型（应该是指向切片的指针或结构体）
		destValue := reflect.ValueOf(dest[index])
		kind := destValue.Elem().Kind()
		if destValue.Kind() != reflect.Ptr || (kind != reflect.Slice && kind != reflect.Struct) {
			return fmt.Errorf("dest[%d] must be a pointer to a struct or slice", index)
		}

		// 获取查询结果的列名
		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns: %v", err)
		}

		// 获取字段映射缓存
		sliceElemType := destValue.Elem().Type()
		if kind == reflect.Slice {
			sliceElemType = sliceElemType.Elem()
		}
		fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)

		// 创建扫描目标数组，用于存放每一列的值
		scanDest := make([]any, len(columns))
		for i := range columns {
			// 判断字段是否存在映射
			if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
				// 获取结构体字段类型
				fieldType := sliceElemType.Field(fieldIndex).Type
				scanDest[i] = reflect.New(fieldType).Interface()
			}
		}

		// 判断目标类型是切片还是结构体
		if kind == reflect.Slice {
			// 预分配内存，创建一个空的目标切片
			results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)

			// 遍历每一行结果
			for rows.Next() {
				// 创建新的结构体实例
				newItem := reflect.New(sliceElemType).Elem()

				// 执行扫描，将当前行数据扫描到 scanDest 数组中
				if err := rows.Scan(scanDest...); err != nil {
					return fmt.Errorf("structure and query fields do not match: %v", err)
				}

				// 将扫描到的数据填充到结构体字段
				for i, col := range columns {
					// 判断列名是否有对应的字段映射
					if fieldIndex, ok := fieldsMapping[col]; ok {
						// 获取结构体中对应字段
						field := newItem.Field(fieldIndex)
						// 获取扫描到的值
						value := reflect.ValueOf(scanDest[i]).Elem()
						// 如果字段不是指针类型，直接赋值
						field.Set(value)
					}
				}

				// 将当前结构体添加到目标切片中
				results = reflect.Append(results, newItem)
			}

			// 设置目标切片为查询结果
			destValue.Elem().Set(results)
		} else {
			// 如果目标是结构体，则只读取一行数据
			if rows.Next() {
				// 创建新的结构体实例
				newItem := reflect.New(sliceElemType).Elem()

				// 执行扫描，将当前行数据扫描到 scanDest 数组中
				if err := rows.Scan(scanDest...); err != nil {
					return fmt.Errorf("structure and query fields do not match: %v", err)
				}

				// 将扫描到的数据填充到结构体字段
				for i, col := range columns {
					// 判断列名是否有对应的字段映射
					if fieldIndex, ok := fieldsMapping[col]; ok {
						// 获取结构体中对应字段
						field := newItem.Field(fieldIndex)
						// 获取扫描到的值
						value := reflect.ValueOf(scanDest[i]).Elem()

						// 如果字段不是指针类型，直接赋值
						field.Set(value)
					}
				}

				// 将数据填充到目标结构体
				destValue.Elem().Set(newItem)
			}

		}

		// 检查是否还有下一个结果集，如果没有，则退出循环
		if !rows.NextResultSet() {
			break
		}
	}

	// 检查行迭代中的错误
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %v", err)
	}

	return nil
}

// 执行查询并将结果映射到结构体中，查询一条数据
func Exec(query string, args ...any) (bool, error) {
	mysql_client.debugLog(query, args...)

	// 使用预处理来避免重复解析 SQL 查询
	stmt, err := mysql_client.DB.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare query: %v", err)
	}
	defer stmt.Close()

	// 执行
	result, err := stmt.Exec(args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %v", err)
	}

	// 获取受影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %v", err)
	}

	// 检查是否有行被更新
	if rowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

// Close 关闭数据库连接
func Close() error {
	return mysql_client.DB.Close()
}
