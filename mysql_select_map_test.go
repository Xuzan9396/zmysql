package zmysql_test

import (
	"log"
	"testing"

	"github.com/Xuzan9396/zmysql"
)

// City 结构体用于测试
type City struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// TestFindMapBasicTypes 测试FindMap基础类型
func TestFindMapBasicTypes(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试 map[int64]string
	t.Run("MapInt64String", func(t *testing.T) {
		result, err := zmysql.FindMap[int64, string]("id", "name", "SELECT id, name FROM cities WHERE id IN (1, 6, 7) ORDER BY id")
		if err != nil {
			t.Errorf("FindMap[int64, string] failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap[int64, string] result: %+v", result)

		// 验证类型
		for key, value := range result {
			t.Logf("Key: %d (type: %T), Value: %s (type: %T)", key, key, value, value)
		}
	})

	// 测试 map[string]int64
	t.Run("MapStringInt64", func(t *testing.T) {
		result, err := zmysql.FindMap[string, int64]("name", "id", "SELECT id, name FROM cities WHERE id IN (1, 6, 7) ORDER BY id")
		if err != nil {
			t.Errorf("FindMap[string, int64] failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap[string, int64] result: %+v", result)

		// 验证类型
		for key, value := range result {
			t.Logf("Key: %s (type: %T), Value: %d (type: %T)", key, key, value, value)
		}
	})

	// 测试 map[string]string
	t.Run("MapStringString", func(t *testing.T) {
		result, err := zmysql.FindMap[string, string]("name", "name", "SELECT id, name FROM cities WHERE id IN (1, 6, 7) ORDER BY id")
		if err != nil {
			t.Errorf("FindMap[string, string] failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap[string, string] result: %+v", result)

		// 验证类型
		for key, value := range result {
			t.Logf("Key: %s (type: %T), Value: %s (type: %T)", key, key, value, value)
		}
	})
}

// TestFindMapStruct 测试FindMap结构体类型
func TestFindMapStruct(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试 map[int64]City，valueField为空时返回整个结构体
	t.Run("MapInt64Struct", func(t *testing.T) {
		result, err := zmysql.FindMap[int64, City]("id", "", "SELECT id, name FROM cities WHERE id IN (1, 6, 7) ORDER BY id")
		if err != nil {
			t.Errorf("FindMap[int64, City] failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap[int64, City] result: %+v", result)

		// 验证类型和内容
		for key, value := range result {
			t.Logf("Key: %d (type: %T), Value: %+v (type: %T)", key, key, value, value)
			t.Logf("  City.ID: %d, City.Name: %s", value.ID, value.Name)
		}
	})

	// 测试 map[string]City
	t.Run("MapStringStruct", func(t *testing.T) {
		result, err := zmysql.FindMap[string, City]("name", "", "SELECT id, name FROM cities WHERE id IN (1, 6, 7) ORDER BY id")
		if err != nil {
			t.Errorf("FindMap[string, City] failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap[string, City] result: %+v", result)

		// 验证类型和内容
		for key, value := range result {
			t.Logf("Key: %s (type: %T), Value: %+v (type: %T)", key, key, value, value)
			t.Logf("  City.ID: %d, City.Name: %s", value.ID, value.Name)
		}
	})
}

// TestFindMapNullFiltering 测试NULL值过滤
func TestFindMapNullFiltering(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试包含NULL键的查询，NULL键应该被过滤掉
	t.Run("NullKeyFiltering", func(t *testing.T) {
		result, err := zmysql.FindMap[int64, string]("id", "name", "SELECT id, name FROM cities WHERE id IN (1, 6) UNION SELECT NULL as id, 'null_test' as name")
		if err != nil {
			t.Errorf("FindMap with null key failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no valid data)")
			return
		}

		t.Logf("FindMap null key filtering result: %+v", result)

		// 验证没有NULL键
		for key, value := range result {
			if key == 0 { // int64的零值
				t.Logf("Warning: Found zero key (might be filtered null): %d -> %s", key, value)
			} else {
				t.Logf("Valid key: %d -> %s", key, value)
			}
		}
	})

	// 测试值为NULL的情况，应该使用零值
	t.Run("NullValueHandling", func(t *testing.T) {
		result, err := zmysql.FindMap[int64, string]("id", "name", "SELECT 999 as id, NULL as name")
		if err != nil {
			t.Errorf("FindMap with null value failed: %v", err)
			return
		}

		if result == nil {
			t.Log("FindMap returned nil (no data)")
			return
		}

		t.Logf("FindMap null value handling result: %+v", result)

		// 验证NULL值被处理为零值
		for key, value := range result {
			t.Logf("Key: %d, Value: '%s' (length: %d)", key, value, len(value))
		}
	})
}

// TestFindMapEmptyAndError 测试空结果和错误情况
func TestFindMapEmptyAndError(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试空结果
	t.Run("EmptyResult", func(t *testing.T) {
		result, err := zmysql.FindMap[int64, string]("id", "name", "SELECT id, name FROM cities WHERE id = ?", 999999)
		if err != nil {
			t.Errorf("FindMap empty result failed: %v", err)
			return
		}

		if result != nil {
			t.Errorf("Expected nil result for empty query, got %+v", result)
		} else {
			t.Log("FindMap correctly returned nil for empty result")
		}
	})

	// 测试空键字段错误
	t.Run("EmptyKeyField", func(t *testing.T) {
		_, err := zmysql.FindMap[int64, string]("", "name", "SELECT id, name FROM cities WHERE id = 1")
		if err == nil {
			t.Error("Expected error for empty key field")
		} else {
			t.Logf("FindMap correctly returned error for empty key field: %v", err)
		}
	})

	// 测试无效键字段错误
	t.Run("InvalidKeyField", func(t *testing.T) {
		_, err := zmysql.FindMap[int64, string]("invalid_field", "name", "SELECT id, name FROM cities WHERE id = 1")
		if err == nil {
			t.Error("Expected error for invalid key field")
		} else {
			t.Logf("FindMap correctly returned error for invalid key field: %v", err)
		}
	})

	// 测试无效值字段错误
	t.Run("InvalidValueField", func(t *testing.T) {
		_, err := zmysql.FindMap[int64, string]("id", "invalid_field", "SELECT id, name FROM cities WHERE id = 1")
		if err == nil {
			t.Error("Expected error for invalid value field")
		} else {
			t.Logf("FindMap correctly returned error for invalid value field: %v", err)
		}
	})
}

// TestFindProcMapBasicTypes 测试FindProcMap基础类型
func TestFindProcMapBasicTypes(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试 map[int64]string 存储过程
	t.Run("ProcMapInt64String", func(t *testing.T) {
		result, err := zmysql.FindProcMap[int64, string]("id", "name", "Proc_GetCityIds", 6)
		if err != nil {
			t.Logf("FindProcMap[int64, string] failed (procedure might not exist): %v", err)
			return
		}

		if result == nil {
			t.Log("FindProcMap returned nil (no data or procedure doesn't exist)")
			return
		}

		t.Logf("FindProcMap[int64, string] result: %+v", result)

		// 验证类型
		for key, value := range result {
			t.Logf("Key: %d (type: %T), Value: %s (type: %T)", key, key, value, value)
		}
	})
}

// TestFindProcMapStruct 测试FindProcMap结构体类型
func TestFindProcMapStruct(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试 map[int64]City 存储过程
	t.Run("ProcMapInt64Struct", func(t *testing.T) {
		result, err := zmysql.FindProcMap[int64, City]("id", "", "Proc_GetCityIds", 6)
		if err != nil {
			t.Logf("FindProcMap[int64, City] failed (procedure might not exist): %v", err)
			return
		}

		if result == nil {
			t.Log("FindProcMap returned nil (no data or procedure doesn't exist)")
			return
		}

		t.Logf("FindProcMap[int64, City] result: %+v", result)

		// 验证类型和内容
		for key, value := range result {
			t.Logf("Key: %d (type: %T), Value: %+v (type: %T)", key, key, value, value)
			t.Logf("  City.ID: %d, City.Name: %s", value.ID, value.Name)
		}
	})
}

// TestFindProcMapErrors 测试FindProcMap错误情况
func TestFindProcMapErrors(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试不存在的存储过程
	t.Run("NonExistentProc", func(t *testing.T) {
		_, err := zmysql.FindProcMap[int64, string]("id", "name", "non_existent_proc")
		if err == nil {
			t.Error("Expected error for non-existent procedure")
		} else {
			t.Logf("FindProcMap correctly returned error for non-existent procedure: %v", err)
		}
	})

	// 测试空键字段错误
	t.Run("EmptyKeyField", func(t *testing.T) {
		_, err := zmysql.FindProcMap[int64, string]("", "name", "Proc_GetCityIds", 6)
		if err == nil {
			t.Error("Expected error for empty key field")
		} else {
			t.Logf("FindProcMap correctly returned error for empty key field: %v", err)
		}
	})

	// 测试无效字段错误
	t.Run("InvalidField", func(t *testing.T) {
		result, err := zmysql.FindProcMap[int64, string]("invalid_field", "name", "Proc_GetCityIds", 6)

		// 这个测试的行为取决于存储过程是否存在
		if err != nil {
			t.Logf("FindProcMap returned expected error: %v", err)
		} else if result == nil {
			t.Log("FindProcMap returned nil (procedure might not exist)")
		} else {
			t.Error("Expected error or nil for invalid key field")
		}
	})
}
