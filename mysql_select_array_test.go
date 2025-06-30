package zmysql_test

import (
	"log"
	"testing"

	"github.com/Xuzan9396/zmysql"
)

// TestFindArrayInt64 测试FindArrayInt64方法
func TestFindArrayInt64(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况
	ids, err := zmysql.FindArrayInt64("id", "SELECT id, name FROM cities WHERE id IN (1, 6,7,8) ORDER BY id")
	if err != nil {
		t.Fatalf("error querying city ids: %v", err)
	}
	t.Logf("有数据测试 - ids: %v, length: %d", ids, len(ids))

	// 测试无数据的情况
	ids, err = zmysql.FindArrayInt64("id", "SELECT id, name FROM cities WHERE id = ?", 999999)
	if err != nil {
		t.Fatalf("error querying non-existent city ids: %v", err)
	}
	if ids == nil {
		t.Log("无数据测试 - 返回 nil")
	} else {
		t.Logf("无数据测试 - ids: %v, length: %d", ids, len(ids))
	}

	// 测试包含NULL值的情况
	ids, err = zmysql.FindArrayInt64("id", "SELECT id, name FROM cities WHERE id IN (1, 6) UNION SELECT NULL as id, 'test' as name")
	if err != nil {
		t.Fatalf("error querying ids with null: %v", err)
	}
	t.Logf("包含NULL值测试 - ids: %v, length: %d", ids, len(ids))

	// 测试字段不存在的情况
	ids, err = zmysql.FindArrayInt64("non_existent_field", "SELECT id, name FROM cities WHERE id = 1")
	if err != nil {
		t.Logf("字段不存在测试 - 期望的错误: %v", err)
	} else {
		t.Error("字段不存在但没有返回错误")
	}
}

// TestFindArrayString 测试FindArrayString方法
func TestFindArrayString(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况
	names, err := zmysql.FindArrayString("name", "SELECT id, name FROM cities WHERE id IN (1, 6,7) ORDER BY id")
	if err != nil {
		t.Fatalf("error querying city names: %v", err)
	}
	t.Logf("有数据测试 - names: %v, length: %d", names, len(names))

	// 测试无数据的情况
	names, err = zmysql.FindArrayString("name", "SELECT id, name FROM cities WHERE id = ?", 999999)
	if err != nil {
		t.Fatalf("error querying non-existent city names: %v", err)
	}
	if names == nil {
		t.Log("无数据测试 - 返回 nil")
	} else {
		t.Logf("无数据测试 - names: %v, length: %d", names, len(names))
	}

	// 测试包含NULL值和空字符串的情况
	names, err = zmysql.FindArrayString("name", "SELECT 'test1' as name UNION SELECT NULL as name UNION SELECT '' as name")
	if err != nil {
		t.Fatalf("error querying names with null and empty: %v", err)
	}
	t.Logf("包含NULL值和空字符串测试 - names: %v, length: %d", names, len(names))

	// 测试字段不存在的情况
	names, err = zmysql.FindArrayString("non_existent_field", "SELECT id, name FROM cities WHERE id = 1")
	if err != nil {
		t.Logf("字段不存在测试 - 期望的错误: %v", err)
	} else {
		t.Error("字段不存在但没有返回错误")
	}
}

// TestFindProcArrayInt64 测试FindProcArrayInt64方法
func TestFindProcArrayInt64(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况 - 假设存在名为 Proc_GetCityIds 的存储过程
	ids, err := zmysql.FindProcArrayInt64("id", "Proc_GetCityIds", 6)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程有数据测试 - ids: %v, length: %d", ids, len(ids))
	}

	// 测试无数据的情况
	ids, err = zmysql.FindProcArrayInt64("id", "Proc_GetCityIds", 999999)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		if ids == nil {
			t.Log("存储过程无数据测试 - 返回 nil")
		} else {
			t.Logf("存储过程无数据测试 - ids: %v, length: %d", ids, len(ids))
		}
	}

	// 测试字段不存在的情况
	ids, err = zmysql.FindProcArrayInt64("non_existent_field", "Proc_GetCityIds", 1)
	if err != nil {
		t.Logf("存储过程字段不存在测试 - 期望的错误: %v", err)
	} else {
		t.Error("存储过程字段不存在但没有返回错误")
	}
}

// TestFindProcArrayString 测试FindProcArrayString方法
func TestFindProcArrayString(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况 - 假设存在名为 Proc_GetCityNames 的存储过程
	names, err := zmysql.FindProcArrayString("name", "Proc_GetCityNames", 6)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程有数据测试 - names: %v, length: %d", names, len(names))
	}

	// 测试无数据的情况
	names, err = zmysql.FindProcArrayString("name", "Proc_GetCityNames", 999999)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		if names == nil {
			t.Log("存储过程无数据测试 - 返回 nil")
		} else {
			t.Logf("存储过程无数据测试 - names: %v, length: %d", names, len(names))
		}
	}

	// 测试字段不存在的情况
	names, err = zmysql.FindProcArrayString("non_existent_field", "Proc_GetCityNames", 1)
	if err != nil {
		t.Logf("存储过程字段不存在测试 - 期望的错误: %v", err)
	} else {
		t.Error("存储过程字段不存在但没有返回错误")
	}
}

// TestFindArrayGeneric 测试泛型FindArray方法
func TestFindArrayGeneric(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试泛型int64数组
	ids, err := zmysql.FindArray[int64]("id", "SELECT id, name FROM cities WHERE id IN (1, 6,7) ORDER BY id")
	if err != nil {
		t.Fatalf("error querying city ids with generic: %v", err)
	}
	t.Logf("泛型int64测试 - ids: %v, length: %d", ids, len(ids))

	// 测试泛型string数组
	names, err := zmysql.FindArray[string]("name", "SELECT id, name FROM cities WHERE id IN (1, 6) ORDER BY id")
	if err != nil {
		t.Fatalf("error querying city names with generic: %v", err)
	}
	t.Logf("泛型string测试 - names: %v, length: %d", names, len(names))

	// 测试包含NULL值的情况
	ids, err = zmysql.FindArray[int64]("id", "SELECT id, name FROM cities WHERE id IN (1, 6) UNION SELECT NULL as id, 'test' as name")
	if err != nil {
		t.Fatalf("error querying ids with null using generic: %v", err)
	}
	t.Logf("泛型NULL值过滤测试 - ids: %v, length: %d", ids, len(ids))

	// 测试无数据的情况
	emptyIds, err := zmysql.FindArray[int64]("id", "SELECT id, name FROM cities WHERE id = ?", 999999)
	if err != nil {
		t.Fatalf("error querying non-existent city ids with generic: %v", err)
	}
	if emptyIds == nil {
		t.Log("泛型无数据测试 - 返回 nil")
	} else {
		t.Logf("泛型无数据测试 - ids: %v, length: %d", emptyIds, len(emptyIds))
	}
}

// TestFindProcArrayGeneric 测试泛型FindProcArray方法
func TestFindProcArrayGeneric(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试泛型存储过程int64数组
	ids, err := zmysql.FindProcArray[int64]("id", "Proc_GetCityIds", 6)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程int64测试 - ids: %v, length: %d", ids, len(ids))
	}

	// 测试泛型存储过程string数组
	names, err := zmysql.FindProcArray[string]("name", "Proc_GetCityNames", 6)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程string测试 - names: %v, length: %d", names, len(names))
	}

	// 测试无数据的情况
	emptyIds, err := zmysql.FindProcArray[int64]("id", "Proc_GetCityIds", 999999)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		if emptyIds == nil {
			t.Log("泛型存储过程无数据测试 - 返回 nil")
		} else {
			t.Logf("泛型存储过程无数据测试 - ids: %v, length: %d", emptyIds, len(emptyIds))
		}
	}
}

// TestFirstColAnyGeneric 测试泛型FirstColAny方法
func TestFirstColAnyGeneric(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试泛型int64单列查询
	id, hasData, err := zmysql.FirstColAny[int64]("SELECT id FROM cities WHERE id = ? LIMIT 1", 6)
	if err != nil {
		t.Fatalf("error querying city id with generic: %v", err)
	}
	t.Logf("泛型int64单列测试 - id: %d, hasData: %t", id, hasData)

	// 测试泛型string单列查询
	name, hasData, err := zmysql.FirstColAny[string]("SELECT name FROM cities WHERE id = ? LIMIT 1", 6)
	if err != nil {
		t.Fatalf("error querying city name with generic: %v", err)
	}
	t.Logf("泛型string单列测试 - name: %s, hasData: %t", name, hasData)

	// 测试无数据的情况
	emptyId, hasData, err := zmysql.FirstColAny[int64]("SELECT id FROM cities WHERE id = ? LIMIT 1", 999999)
	if err != nil {
		t.Fatalf("error querying non-existent city id with generic: %v", err)
	}
	t.Logf("泛型无数据测试 - id: %d, hasData: %t", emptyId, hasData)

	// 测试NULL值的情况
	nullId, hasData, err := zmysql.FirstColAny[int64]("SELECT NULL as id LIMIT 1")
	if err != nil {
		t.Fatalf("error querying null id with generic: %v", err)
	}
	t.Logf("泛型NULL值测试 - id: %d, hasData: %t", nullId, hasData)
}

// TestFirstColProcAnyGeneric 测试泛型FirstColProcAny方法
func TestFirstColProcAnyGeneric(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试泛型存储过程int64单列查询
	id, hasData, err := zmysql.FirstColProcAny[int64]("Proc_GetCityId", 6)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程int64测试 - id: %d, hasData: %t", id, hasData)
	}

	// 测试泛型存储过程string单列查询
	name, hasData, err := zmysql.FirstColProcAny[string]("Proc_GetCityName", 6)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程string测试 - name: %s, hasData: %t", name, hasData)
	}

	// 测试无数据的情况
	emptyId, hasData, err := zmysql.FirstColProcAny[int64]("Proc_GetCityId", 999999)
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程无数据测试 - id: %d, hasData: %t", emptyId, hasData)
	}

	// 测试NULL值的情况
	nullId, hasData, err := zmysql.FirstColProcAny[int64]("Proc_GetNull")
	if err != nil {
		t.Logf("泛型存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("泛型存储过程NULL值测试 - id: %d, hasData: %t", nullId, hasData)
	}
}
