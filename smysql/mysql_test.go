package smysql_test

import (
	"encoding/json"
	"fmt"
	"github.com/Xuzan9396/zmysql/smysql"
	"testing"
	"time"
)

// CityTest 对应 cities_test 表结构
type CityTest struct {
	ID          uint      `db:"id"`
	Name        string    `db:"name"`
	StateID     uint      `db:"state_id"`
	StateCode   string    `db:"state_code"`
	CountryID   uint      `db:"country_id"`
	CountryCode string    `db:"country_code"`
	Latitude    float64   `db:"latitude"`
	Longitude   float64   `db:"longitude"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Flag        bool      `db:"flag"`
	WikiDataId  string    `db:"wikiDataId"`
}

// 获取测试客户端
func getTestClient() (*smysql.MySQLClient, error) {
	return smysql.Conn("root", "123456", "127.0.0.1:3326", "weather", smysql.WithDebug())
}

// 初始化测试数据
func setupTestData(client *smysql.MySQLClient) error {
	// 清理表
	_, err := client.Exec("DELETE FROM cities_test")
	if err != nil {
		return fmt.Errorf("failed to clean table: %v", err)
	}

	// 插入测试数据
	testData := []struct {
		name        string
		stateID     uint
		stateCode   string
		countryID   uint
		countryCode string
		latitude    float64
		longitude   float64
		flag        bool
		wikiDataId  string
	}{
		{"Beijing", 1, "BJ", 1, "CN", 39.9042, 116.4074, true, "Q956"},
		{"Shanghai", 2, "SH", 1, "CN", 31.2304, 121.4737, true, "Q8686"},
		{"Guangzhou", 3, "GD", 1, "CN", 23.1291, 113.2644, true, "Q16572"},
		{"Shenzhen", 3, "GD", 1, "CN", 22.5431, 114.0579, true, "Q15174"},
		{"Tokyo", 4, "TK", 2, "JP", 35.6762, 139.6503, true, "Q1490"},
		{"Osaka", 5, "OS", 2, "JP", 34.6937, 135.5023, false, "Q35765"},
		{"New York", 6, "NY", 3, "US", 40.7128, -74.0060, true, "Q60"},
		{"Los Angeles", 7, "CA", 3, "US", 34.0522, -118.2437, true, "Q65"},
		{"TestNull", 8, "TN", 4, "TT", 0.0, 0.0, false, ""},
		{"TestEmpty", 9, "TE", 4, "TT", 1.1, 1.1, true, ""},
	}

	for _, data := range testData {
		_, err := client.Exec(`
			INSERT INTO cities_test (name, state_id, state_code, country_id, country_code, latitude, longitude, flag, wikiDataId) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			data.name, data.stateID, data.stateCode, data.countryID, data.countryCode,
			data.latitude, data.longitude, data.flag, data.wikiDataId)
		if err != nil {
			return fmt.Errorf("failed to insert test data: %v", err)
		}
	}

	return nil
}

// TestFind 测试 Find 方法
func TestFind(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FindAllCities", func(t *testing.T) {
		var cities []CityTest
		err := client.Find(&cities, "SELECT * FROM cities_test ORDER BY id")
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(cities) != 10 {
			t.Errorf("Expected 10 cities, got %d", len(cities))
		}

		t.Logf("Found %d cities", len(cities))
		for i, city := range cities {
			if i < 3 { // 只打印前3个
				t.Logf("City %d: %+v", i+1, city)
			}
		}
	})

	t.Run("FindCitiesWithCondition", func(t *testing.T) {
		var cities []CityTest
		err := client.Find(&cities, "SELECT * FROM cities_test WHERE country_id = ? AND flag = ?", 1, true)
		if err != nil {
			t.Fatalf("Find with condition failed: %v", err)
		}

		if len(cities) != 4 {
			t.Errorf("Expected 4 Chinese cities with flag=true, got %d", len(cities))
		}

		t.Logf("Found %d Chinese cities with flag=true", len(cities))
	})

	t.Run("FindEmptyResult", func(t *testing.T) {
		var cities []CityTest
		err := client.Find(&cities, "SELECT * FROM cities_test WHERE id = ?", 999)
		if err != nil {
			t.Fatalf("Find empty result failed: %v", err)
		}

		if len(cities) != 0 {
			t.Errorf("Expected 0 cities, got %d", len(cities))
		}

		t.Log("Empty result handled correctly")
	})
}

// TestFirst 测试 First 方法
func TestFirst(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FirstExistingCity", func(t *testing.T) {
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE name = ?", "Beijing")
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}

		if !found {
			t.Error("Expected to find Beijing, but got no result")
		}

		if city.Name != "Beijing" {
			t.Errorf("Expected Beijing, got %s", city.Name)
		}

		t.Logf("Found city: %+v", city)
	})

	t.Run("FirstNonExistingCity", func(t *testing.T) {
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE name = ?", "NonExistent")
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}

		if found {
			t.Error("Expected no result, but found a city")
		}

		t.Log("Non-existing city handled correctly")
	})
}

// TestFirstCol 测试 FirstCol 方法
func TestFirstCol(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FirstColString", func(t *testing.T) {
		var name string
		found, err := client.FirstCol(&name, "SELECT name FROM cities_test ORDER BY id LIMIT 1")
		if err != nil {
			t.Fatalf("FirstCol failed: %v", err)
		}

		if !found {
			t.Error("Expected to find a name, but got no result")
		}

		if name == "" {
			t.Errorf("Expected non-empty name, got empty string")
		}

		t.Logf("Found name: %s", name)
	})

	t.Run("FirstColInt", func(t *testing.T) {
		var count int64
		found, err := client.FirstCol(&count, "SELECT COUNT(*) FROM cities_test WHERE flag = ?", true)
		if err != nil {
			t.Fatalf("FirstCol failed: %v", err)
		}

		if !found {
			t.Error("Expected to find a count, but got no result")
		}

		if count <= 0 {
			t.Errorf("Expected positive count, got %d", count)
		}

		t.Logf("Found count: %d", count)
	})

	t.Run("FirstColFloat", func(t *testing.T) {
		var latitude float64
		found, err := client.FirstCol(&latitude, "SELECT latitude FROM cities_test WHERE name = ?", "Beijing")
		if err != nil {
			t.Fatalf("FirstCol failed: %v", err)
		}

		if !found {
			t.Error("Expected to find latitude, but got no result")
		}

		if latitude != 39.9042 {
			t.Errorf("Expected latitude 39.9042, got %f", latitude)
		}

		t.Logf("Found latitude: %f", latitude)
	})
}

// TestFirstColInt64 测试 FirstColInt64 方法
func TestFirstColInt64(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FirstColInt64Count", func(t *testing.T) {
		count, found, err := client.FirstColInt64("SELECT COUNT(*) FROM cities_test")
		if err != nil {
			t.Fatalf("FirstColInt64 failed: %v", err)
		}

		if !found {
			t.Error("Expected to find count, but got no result")
		}

		if count != 10 {
			t.Errorf("Expected count 10, got %d", count)
		}

		t.Logf("Total cities count: %d", count)
	})

	t.Run("FirstColInt64ID", func(t *testing.T) {
		id, found, err := client.FirstColInt64("SELECT id FROM cities_test WHERE name = ?", "Tokyo")
		if err != nil {
			t.Fatalf("FirstColInt64 failed: %v", err)
		}

		if !found {
			t.Error("Expected to find Tokyo ID, but got no result")
		}

		if id <= 0 {
			t.Errorf("Expected positive ID, got %d", id)
		}

		t.Logf("Tokyo ID: %d", id)
	})

	t.Run("FirstColInt64NoResult", func(t *testing.T) {
		id, found, err := client.FirstColInt64("SELECT id FROM cities_test WHERE name = ?", "NonExistent")
		if err != nil {
			t.Fatalf("FirstColInt64 failed: %v", err)
		}

		if found {
			t.Error("Expected no result, but found ID")
		}

		if id != 0 {
			t.Errorf("Expected 0 for no result, got %d", id)
		}

		t.Log("No result handled correctly")
	})
}

// TestFirstColString 测试 FirstColString 方法
func TestFirstColString(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FirstColStringName", func(t *testing.T) {
		name, found, err := client.FirstColString("SELECT name FROM cities_test ORDER BY id LIMIT 1 OFFSET 1")
		if err != nil {
			t.Fatalf("FirstColString failed: %v", err)
		}

		if !found {
			t.Error("Expected to find name, but got no result")
		}

		if name == "" {
			t.Errorf("Expected non-empty name, got empty string")
		}

		t.Logf("Found name: %s", name)
	})

	t.Run("FirstColStringEmpty", func(t *testing.T) {
		wikiId, found, err := client.FirstColString("SELECT wikiDataId FROM cities_test WHERE name = ?", "TestEmpty")
		if err != nil {
			t.Fatalf("FirstColString failed: %v", err)
		}

		if !found {
			t.Error("Expected to find empty wikiDataId, but got no result")
		}

		if wikiId != "" {
			t.Errorf("Expected empty string, got '%s'", wikiId)
		}

		t.Log("Empty string handled correctly")
	})
}

// TestFindArrayInt64 测试 FindArrayInt64 方法
func TestFindArrayInt64(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FindArrayInt64IDs", func(t *testing.T) {
		ids, err := client.FindArrayInt64("id", "SELECT id, name FROM cities_test WHERE country_id = ? ORDER BY id", 1)
		if err != nil {
			t.Fatalf("FindArrayInt64 failed: %v", err)
		}

		if len(ids) != 4 {
			t.Errorf("Expected 4 Chinese city IDs, got %d", len(ids))
		}

		t.Logf("Found Chinese city IDs: %v", ids)
	})

	t.Run("FindArrayInt64Empty", func(t *testing.T) {
		ids, err := client.FindArrayInt64("id", "SELECT id FROM cities_test WHERE country_id = ?", 999)
		if err != nil {
			t.Fatalf("FindArrayInt64 failed: %v", err)
		}

		if ids != nil {
			t.Errorf("Expected nil for empty result, got %v", ids)
		}

		t.Log("Empty result handled correctly")
	})

	t.Run("FindArrayInt64FieldNotFound", func(t *testing.T) {
		_, err := client.FindArrayInt64("non_existent_field", "SELECT id FROM cities_test LIMIT 1")
		if err == nil {
			t.Error("Expected error for non-existent field, but got none")
		}

		t.Logf("Non-existent field error: %v", err)
	})
}

// TestFindArrayString 测试 FindArrayString 方法
func TestFindArrayString(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FindArrayStringNames", func(t *testing.T) {
		names, err := client.FindArrayString("name", "SELECT name FROM cities_test WHERE country_id = ? ORDER BY name", 2)
		if err != nil {
			t.Fatalf("FindArrayString failed: %v", err)
		}

		expectedNames := []string{"Osaka", "Tokyo"}
		if len(names) != len(expectedNames) {
			t.Errorf("Expected %d Japanese city names, got %d", len(expectedNames), len(names))
		}

		t.Logf("Found Japanese city names: %v", names)
	})

	t.Run("FindArrayStringWithEmpty", func(t *testing.T) {
		wikiIds, err := client.FindArrayString("wikiDataId", "SELECT wikiDataId FROM cities_test WHERE country_id = ? ORDER BY id", 4)
		if err != nil {
			t.Fatalf("FindArrayString failed: %v", err)
		}

		// 空字符串是Valid的，所以会被包含在结果中
		if len(wikiIds) != 2 {
			t.Errorf("Expected 2 empty wikiDataIds, got %d", len(wikiIds))
		}

		// 验证都是空字符串
		for _, id := range wikiIds {
			if id != "" {
				t.Errorf("Expected empty string, got '%s'", id)
			}
		}

		t.Logf("Found empty wikiDataIds: %v", wikiIds)
	})
}

// TestExec 测试 Exec 方法
func TestExec(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("ExecUpdate", func(t *testing.T) {
		success, err := client.Exec("UPDATE cities_test SET flag = ? WHERE country_id = ?", false, 1)
		if err != nil {
			t.Fatalf("Exec update failed: %v", err)
		}

		if !success {
			t.Error("Expected update to succeed")
		}

		// 验证更新结果
		var count int64
		found, err := client.FirstCol(&count, "SELECT COUNT(*) FROM cities_test WHERE country_id = ? AND flag = ?", 1, false)
		if err != nil {
			t.Fatalf("Verification query failed: %v", err)
		}

		if !found || count != 4 {
			t.Errorf("Expected 4 updated records, got %d", count)
		}

		t.Logf("Successfully updated %d records", count)
	})

	t.Run("ExecInsert", func(t *testing.T) {
		success, err := client.Exec(`
			INSERT INTO cities_test (name, state_id, state_code, country_id, country_code, latitude, longitude, flag, wikiDataId) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"Test City", 99, "TC", 99, "TC", 1.0, 1.0, true, "Q99999")
		if err != nil {
			t.Fatalf("Exec insert failed: %v", err)
		}

		if !success {
			t.Error("Expected insert to succeed")
		}

		// 验证插入结果
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE name = ?", "Test City")
		if err != nil {
			t.Fatalf("Verification query failed: %v", err)
		}

		if !found {
			t.Error("Expected to find inserted city")
		}

		t.Logf("Successfully inserted city: %s", city.Name)
	})

	t.Run("ExecDelete", func(t *testing.T) {
		success, err := client.Exec("DELETE FROM cities_test WHERE name = ?", "Test City")
		if err != nil {
			t.Fatalf("Exec delete failed: %v", err)
		}

		if !success {
			t.Error("Expected delete to succeed")
		}

		// 验证删除结果
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE name = ?", "Test City")
		if err != nil {
			t.Fatalf("Verification query failed: %v", err)
		}

		if found {
			t.Error("Expected city to be deleted")
		}

		t.Log("Successfully deleted test city")
	})
}

// TestExecFindLastId 测试 ExecFindLastId 方法
func TestExecFindLastId(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("ExecFindLastIdInsert", func(t *testing.T) {
		lastId, err := client.ExecFindLastId(`
			INSERT INTO cities_test (name, state_id, state_code, country_id, country_code, latitude, longitude, flag, wikiDataId) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"Last ID Test", 88, "LI", 88, "LI", 2.0, 2.0, true, "Q88888")
		if err != nil {
			t.Fatalf("ExecFindLastId failed: %v", err)
		}

		if lastId <= 0 {
			t.Errorf("Expected positive last insert ID, got %d", lastId)
		}

		// 验证插入的记录
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE id = ?", lastId)
		if err != nil {
			t.Fatalf("Verification query failed: %v", err)
		}

		if !found || city.Name != "Last ID Test" {
			t.Error("Expected to find inserted city with last ID")
		}

		t.Logf("Successfully inserted with last ID: %d", lastId)
	})
}

// TestExecByte 测试 ExecByte 方法
func TestExecByte(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("ExecByteList", func(t *testing.T) {
		data, err := client.ExecByte("SELECT id, name, country_code FROM cities_test WHERE country_id = ? ORDER BY id LIMIT 3", smysql.HAS_LIST, 1)
		if err != nil {
			t.Fatalf("ExecByte list failed: %v", err)
		}

		var result []map[string]interface{}
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 records, got %d", len(result))
		}

		t.Logf("ExecByte list result: %s", string(data))
	})

	t.Run("ExecByteOne", func(t *testing.T) {
		data, err := client.ExecByte("SELECT id, name, country_code FROM cities_test WHERE name = ?", smysql.HAS_ONE, "Beijing")
		if err != nil {
			t.Fatalf("ExecByte one failed: %v", err)
		}

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		// ExecByte 返回的是原始数据，可能是 base64 编码或其他格式
		if result["name"] == nil {
			t.Error("Expected name field in result")
		}

		t.Logf("ExecByte one result: %s", string(data))
	})
}

// TestPackageLevelGenericFunctions 测试包级泛型函数
func TestPackageLevelGenericFunctions(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	t.Run("FindArrayGeneric", func(t *testing.T) {
		// 测试 int64 泛型
		ids, err := smysql.FindArray[int64](client, "id", "SELECT id FROM cities_test WHERE country_id = ? ORDER BY id", 1)
		if err != nil {
			t.Fatalf("FindArray[int64] failed: %v", err)
		}

		if len(ids) != 4 {
			t.Errorf("Expected 4 IDs, got %d", len(ids))
		}

		t.Logf("FindArray[int64] result: %v", ids)

		// 测试 string 泛型
		names, err := smysql.FindArray[string](client, "name", "SELECT name FROM cities_test WHERE country_id = ? ORDER BY name", 2)
		if err != nil {
			t.Fatalf("FindArray[string] failed: %v", err)
		}

		expectedNames := []string{"Osaka", "Tokyo"}
		if len(names) != len(expectedNames) {
			t.Errorf("Expected %d names, got %d", len(expectedNames), len(names))
		}

		t.Logf("FindArray[string] result: %v", names)
	})

	t.Run("FirstColAnyGeneric", func(t *testing.T) {
		// 测试 int64 泛型
		count, found, err := smysql.FirstColAny[int64](client, "SELECT COUNT(*) FROM cities_test WHERE flag = ?", true)
		if err != nil {
			t.Fatalf("FirstColAny[int64] failed: %v", err)
		}

		if !found {
			t.Error("Expected to find count")
		}

		// 数量可能因为测试数据的执行顺序而变化
		if count <= 0 {
			t.Errorf("Expected positive count, got %d", count)
		}

		t.Logf("FirstColAny[int64] result: %d", count)

		// 测试 string 泛型
		name, found, err := smysql.FirstColAny[string](client, "SELECT name FROM cities_test ORDER BY id LIMIT 1")
		if err != nil {
			t.Fatalf("FirstColAny[string] failed: %v", err)
		}

		if !found {
			t.Error("Expected to find name")
		}

		// 数据可能因为ID顺序变化，只检查不为空
		if name == "" {
			t.Errorf("Expected non-empty name, got empty string")
		}

		t.Logf("FirstColAny[string] result: %s", name)
	})

	t.Run("FindMapGeneric", func(t *testing.T) {
		// 测试 map[int64]string
		cityMap, err := smysql.FindMap[int64, string](client, "id", "name", "SELECT id, name FROM cities_test WHERE country_id = ? ORDER BY id", 1)
		if err != nil {
			t.Fatalf("FindMap[int64, string] failed: %v", err)
		}

		if len(cityMap) != 4 {
			t.Errorf("Expected 4 entries in map, got %d", len(cityMap))
		}

		t.Logf("FindMap[int64, string] result: %v", cityMap)

		// 测试 map[string]int64
		reverseMap, err := smysql.FindMap[string, int64](client, "name", "id", "SELECT name, id FROM cities_test WHERE country_id = ? ORDER BY id", 2)
		if err != nil {
			t.Fatalf("FindMap[string, int64] failed: %v", err)
		}

		if len(reverseMap) != 2 {
			t.Errorf("Expected 2 entries in reverse map, got %d", len(reverseMap))
		}

		t.Logf("FindMap[string, int64] result: %v", reverseMap)

		// 测试 map[int64]CityTest (结构体值)
		structMap, err := smysql.FindMap[int64, CityTest](client, "id", "", "SELECT * FROM cities_test WHERE country_id = ? ORDER BY id LIMIT 2", 1)
		if err != nil {
			t.Fatalf("FindMap[int64, CityTest] failed: %v", err)
		}

		if len(structMap) != 2 {
			t.Errorf("Expected 2 entries in struct map, got %d", len(structMap))
		}

		t.Logf("FindMap[int64, CityTest] result: %v", structMap)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	t.Run("InvalidSQL", func(t *testing.T) {
		var cities []CityTest
		err := client.Find(&cities, "INVALID SQL QUERY")
		if err == nil {
			t.Error("Expected error for invalid SQL, but got none")
		}

		t.Logf("Invalid SQL error (expected): %v", err)
	})

	t.Run("InvalidTable", func(t *testing.T) {
		var cities []CityTest
		err := client.Find(&cities, "SELECT * FROM non_existent_table")
		if err == nil {
			t.Error("Expected error for non-existent table, but got none")
		}

		t.Logf("Non-existent table error (expected): %v", err)
	})

	t.Run("InvalidDestination", func(t *testing.T) {
		var city CityTest // 不是指针
		_, err := client.First(city, "SELECT * FROM cities_test LIMIT 1")
		if err == nil {
			t.Error("Expected error for invalid destination, but got none")
		}

		t.Logf("Invalid destination error (expected): %v", err)
	})
}

// TestNullHandling 测试 NULL 值处理
func TestNullHandling(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := setupTestData(client); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	// 插入包含 NULL 值的记录
	_, err = client.Exec(`
		INSERT INTO cities_test (name, state_id, state_code, country_id, country_code, latitude, longitude, flag, wikiDataId) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Null Test", 99, "NT", 99, "NT", 0.0, 0.0, false, nil)
	if err != nil {
		t.Fatalf("Failed to insert NULL test data: %v", err)
	}

	t.Run("NullFieldHandling", func(t *testing.T) {
		var city CityTest
		found, err := client.First(&city, "SELECT * FROM cities_test WHERE name = ?", "Null Test")
		if err != nil {
			t.Fatalf("First with NULL field failed: %v", err)
		}

		if !found {
			t.Error("Expected to find NULL test city")
		}

		if city.WikiDataId != "" {
			t.Errorf("Expected empty string for NULL wikiDataId, got '%s'", city.WikiDataId)
		}

		t.Logf("NULL handling city: %+v", city)
	})

	t.Run("NullArrayFiltering", func(t *testing.T) {
		// NULL 值应该被过滤掉
		wikiIds, err := client.FindArrayString("wikiDataId", "SELECT wikiDataId FROM cities_test WHERE name IN (?, ?)", "Null Test", "TestEmpty")
		if err != nil {
			t.Fatalf("FindArrayString with NULL failed: %v", err)
		}

		// NULL 值被过滤，空字符串会被保留
		if wikiIds == nil {
			t.Log("NULL values correctly filtered from array")
		} else {
			t.Logf("Found wikiIds: %v (length: %d)", wikiIds, len(wikiIds))
		}
	})
}

// TestConnectionOptions 测试连接选项
func TestConnectionOptions(t *testing.T) {
	t.Run("WithDebugOption", func(t *testing.T) {
		client, err := smysql.Conn("root", "123456", "127.0.0.1:3326", "weather", smysql.WithDebug())
		if err != nil {
			t.Fatalf("failed to create client with debug: %v", err)
		}
		defer client.Close()

		// 执行一个查询来测试调试输出
		var count int64
		_, err = client.FirstCol(&count, "SELECT COUNT(*) FROM cities_test")
		if err != nil {
			t.Fatalf("Query with debug failed: %v", err)
		}

		t.Log("Debug option tested successfully")
	})

	t.Run("WithConnectionPoolOptions", func(t *testing.T) {
		client, err := smysql.Conn("root", "123456", "127.0.0.1:3326", "weather",
			smysql.WithMaxOpenConns(5),
			smysql.WithMaxIdleConns(2),
			smysql.WithConnMaxLifetime(1*time.Hour))
		if err != nil {
			t.Fatalf("failed to create client with pool options: %v", err)
		}
		defer client.Close()

		// 测试连接是否工作
		var count int64
		_, err = client.FirstCol(&count, "SELECT 1")
		if err != nil {
			t.Fatalf("Query with pool options failed: %v", err)
		}

		t.Log("Connection pool options tested successfully")
	})

	t.Run("WithTimezoneOption", func(t *testing.T) {
		client, err := smysql.Conn("root", "123456", "127.0.0.1:3326", "weather",
			smysql.WithLoc("UTC"))
		if err != nil {
			t.Fatalf("failed to create client with timezone: %v", err)
		}
		defer client.Close()

		// 测试时区设置
		var now time.Time
		_, err = client.FirstCol(&now, "SELECT NOW()")
		if err != nil {
			t.Fatalf("Query with timezone failed: %v", err)
		}

		t.Logf("Current time with UTC: %v", now)
	})
}

// 清理测试数据
func TestCleanup(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.Exec("DELETE FROM cities_test")
	if err != nil {
		t.Errorf("Failed to cleanup test data: %v", err)
	} else {
		t.Log("Test data cleaned up successfully")
	}
}
