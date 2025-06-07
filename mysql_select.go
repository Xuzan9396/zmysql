package zmysql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// createNullScanner 根据字段类型创建对应的Null扫描器，只处理基本类型
func createNullScanner(fieldType reflect.Type) any {
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
		// 对于自定义类型，使用原有方式
		return reflect.New(fieldType).Interface()
	}
}

// setFieldFromNullScanner 从Null扫描器设置字段值
func setFieldFromNullScanner(field reflect.Value, scanner any, fieldType reflect.Type) error {
	switch s := scanner.(type) {
	case *sql.NullString:
		if s.Valid {
			if fieldType.Kind() != reflect.String {
				return fmt.Errorf("cannot convert string to %s", fieldType.Kind())
			}
			field.SetString(s.String)
		} else {
			field.SetString("") // NULL设置为空字符串
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
			// NULL设置为0
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
			field.SetFloat(0.0) // NULL设置为0.0
		}
	case *sql.NullBool:
		if s.Valid {
			if fieldType.Kind() != reflect.Bool {
				return fmt.Errorf("cannot convert bool to %s", fieldType.Kind())
			}
			field.SetBool(s.Bool)
		} else {
			field.SetBool(false) // NULL设置为false
		}
	default:
		// 对于自定义类型，使用原有的设置方式
		value := reflect.ValueOf(scanner).Elem()
		field.Set(value)
	}
	return nil
}

// 通用的扫描逻辑
func scanRows(rows *sql.Rows, destValue reflect.Value, sliceElemType reflect.Type) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)
	results := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)
	scanDest := make([]any, len(columns))

	// 为每列创建合适的Null扫描器
	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := sliceElemType.Field(fieldIndex).Type
			scanDest[i] = createNullScanner(fieldType)
		} else {
			scanDest[i] = &sql.NullString{} // 忽略未定义的字段
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
				if err := setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
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

	// 为每列创建合适的Null扫描器
	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = createNullScanner(fieldType)
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
				if err := setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
					return false, fmt.Errorf("failed to set field %s: %v", col, err)
				}
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

	// 为每列创建合适的Null扫描器
	for i := range columns {
		if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
			fieldType := structType.Field(fieldIndex).Type
			scanDest[i] = createNullScanner(fieldType)
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
				if err := setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
					return false, fmt.Errorf("failed to set field %s: %v", col, err)
				}
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
		// 使用合适的Null扫描器
		scanner := createNullScanner(destValue.Elem().Type())
		if err := rows.Scan(scanner); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		// 设置值
		if err := setFieldFromNullScanner(destValue.Elem(), scanner, destValue.Elem().Type()); err != nil {
			return false, fmt.Errorf("failed to set value: %v", err)
		}
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
		// 使用合适的Null扫描器
		scanner := createNullScanner(destValue.Elem().Type())
		if err := rows.Scan(scanner); err != nil {
			return false, fmt.Errorf("failed to scan column: %v", err)
		}

		// 设置值
		if err := setFieldFromNullScanner(destValue.Elem(), scanner, destValue.Elem().Type()); err != nil {
			return false, fmt.Errorf("failed to set value: %v", err)
		}
		return true, nil
	}

	return false, nil
}

// FindMultipleProc 执行存储过程并将多个结果集映射到多个目标结构体或切片中
func FindMultipleProc(dest []any, procName string, args ...any) error {
	mysql_client.debugLog(procName, args...)

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

		if kind == reflect.Slice {
			// 处理切片类型
			if err := scanRows(rows, destValue, sliceElemType); err != nil {
				return fmt.Errorf("failed to scan result set %d: %v", index, err)
			}
		} else {
			// 处理单个结构体类型
			columns, err := rows.Columns()
			if err != nil {
				return fmt.Errorf("failed to get columns: %v", err)
			}

			fieldsMapping := mysql_client.getFieldsMapping(sliceElemType)
			scanDest := make([]any, len(columns))

			// 为每列创建合适的Null扫描器
			for i := range columns {
				if fieldIndex, ok := fieldsMapping[columns[i]]; ok {
					fieldType := sliceElemType.Field(fieldIndex).Type
					scanDest[i] = createNullScanner(fieldType)
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
						if err := setFieldFromNullScanner(field, scanDest[i], fieldType); err != nil {
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
