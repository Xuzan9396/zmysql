package zmysql_test

import (
	"encoding/json"
	"fmt"
	"github.com/Xuzan9396/zmysql"
	"log"
	"testing"
	"time"
)

type MyTime struct {
	time.Time
}

// Custom MarshalJSON method to format time without timezone
func (t *MyTime) MarshalJSON() ([]byte, error) {
	// Format as "2006-01-02 15:04:05"
	return json.Marshal(t.Time.Format("2006-01-02 15:04:05"))
}

func (t *MyTime) ToString() string {
	// Format as "2006-01-02 15:04:05"
	return t.Time.Format("2006-01-02 15:04:05")
}

// 实现 sql.Scanner 接口，允许从数据库读取时间字段
func (t *MyTime) Scan(value interface{}) error {
	// 将数据库的值转化为 time.Time 类型
	v, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("failed to scan MyTime: %v", value)
	}
	t.Time = v
	return nil
}

type CityInfo struct {
	Name       string  `db:"city_name"`
	ID         int     `db:"id"`
	Latitude   float64 `db:"latitude"`
	Longitude  float64 `db:"longitude"`
	WikiDataId string  `db:"wikiDataId"`
	CreatedAt  MyTime  `json:"create_at" db:"created_at"`
}

func TestQuery(t *testing.T) {
	// 创建 MySQL 客户端
	// 美国时区 America/New_York
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithLoc("America/New_York"), zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 查询数据
	var cities []CityInfo
	err = zmysql.Find(&cities, "SELECT  name as city_name, latitude, longitude,created_at FROM cities WHERE id = ? limit 5", 7)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}
	// 输出每个城市的信息，时间格式化为 "2019-10-05 23:58:06" 格式
	for i, city := range cities {
		t.Log(i, city, city.WikiDataId == "")
	}

	resByte, _ := json.Marshal(cities)
	t.Log(string(resByte))
	t.Log("-----------------\n")
	var cityList []CityInfo
	err = findTest(&cityList)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}
	t.Log("长度", len(cityList))

}

func findTest(dest *[]CityInfo) error {
	return zmysql.Find(dest, "SELECT  name as city_name, latitude, longitude,created_at FROM cities WHERE id = ? limit 5", 1)
}

func TestFirst(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather")
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	type CityInfo struct {
		Name     string  `db:"city_name"`
		ID       int     `db:"id"`
		Latitude float64 `db:"latitude"`
		//Longitude  float64 `db:"longitude"`
		WikiDataId string `db:"wikiDataId"`
	}
	// 查询数据
	var cities CityInfo
	bools, err := zmysql.First(&cities, "SELECT  name as city_name,flag, latitude, longitude FROM cities WHERE id = ? limit 1", 7)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	if !bools {
		t.Log("没有数据")
		return
	}

	t.Log(cities.ID, cities.Name, cities.Latitude, cities.WikiDataId == "")
	resByte, _ := json.Marshal(cities)
	t.Log(string(resByte))
}

func TestMySQLClient_FindMultiple(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather")
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	type CityInfo struct {
		Name       string  `db:"name"`
		ID         int     `db:"id"`
		Haha       float64 `db:"haha"`
		Haha2      string  `db:"haha2"`
		Latitude   float64 `db:"latitude"`
		Longitude  float64 `db:"longitude"`
		WikiDataId string  `db:"wikiDataId"`
	}
	type Ming struct {
		Mingzi string `db:"mingzi"`
	}

	type Total struct {
		Total int `db:"total"`
	}
	// 查询数据
	var cities []CityInfo
	var mings Ming
	var total Total
	err = zmysql.FindMultipleProc([]any{
		&cities,
		&total,
		&mings,
	}, "Proc_FindMultiple")
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	t.Log("长度", len(cities))

	for i, city := range cities {
		t.Log(i, city.ID, city.Name, city.WikiDataId == "", "haha:", city.Haha, "haha2:", city.Haha2)
	}
	t.Log(mings.Mingzi, total.Total)
}

func TestUpdate(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	bools, err := zmysql.Exec("UPDATE cities SET created_at = ?,wikiDataId = ? WHERE id = ?", "2025-02-13 11:49:36", nil, 5)
	if err != nil {
		log.Fatalf("error updating cities: %v", err)
	}
	t.Log(bools)

	bools, err = zmysql.Exec("insert into cities (name, latitude, longitude, created_at,state_id,state_code,country_id,country_code) values (?, ?, ?, ?,?,?,?,?)", "test", 1.1, 1.1, "2025-02-13 11:49:36", 1, "223", 1, "zh")
	if err != nil {
		log.Fatalf("error updating cities: %v", err)
	}
	t.Log(bools)

}

func TestFindCol(t *testing.T) {
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()
	var cityName string
	bools, err := zmysql.FirstCol(&cityName, "SELECT  name as city_name FROM cities WHERE id = ? limit 1", 7)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}
	if bools != true {
		t.Log("没有数据")
		return
	}
	t.Log(cityName)
}

func TestExecByte(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithLoc("America/New_York"), zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 查询数据
	res, err := zmysql.ExecByte("SELECT  name as city_name,state_id, latitude, longitude,created_at FROM cities WHERE id = ? limit 2", zmysql.HAS_LIST, 10)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	t.Log(string(res))

	res, err = zmysql.ExecByte("SELECT  name as city_name,state_id, latitude, longitude,created_at FROM cities WHERE id = ? limit 2", zmysql.HAS_ONE, 10)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	t.Log(string(res))
}

func TestProcByte(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithLoc("America/New_York"), zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 查询数据
	res, err := zmysql.ExecProcByte("Proc_ProcByte", zmysql.HAS_ONE, 5)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	t.Log(string(res))

	res, err = zmysql.ExecProcByte("Proc_ProcByte", zmysql.HAS_LIST, 5)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}

	t.Log(string(res))
}

func TestExecFindLastId(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试插入数据并获取LastInsertId
	lastId, err := zmysql.ExecFindLastId("INSERT INTO cities_test (name, latitude, longitude, created_at, state_id, state_code, country_id, country_code) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"test_city", 12.34, 56.78, "2025-02-13 12:00:00", 1, "TEST", 1, "TC")
	if err != nil {
		log.Fatalf("error inserting city: %v", err)
	}

	t.Logf("插入成功，LastInsertId: %d", lastId)

	// 验证插入的数据
	var city CityInfo
	found, err := zmysql.First(&city, "SELECT id, name as city_name, latitude, longitude, created_at FROM cities WHERE id = ?", lastId)
	if err != nil {
		log.Fatalf("error querying inserted city: %v", err)
	}

	if !found {
		t.Error("插入的数据未找到")
		return
	}

	t.Logf("验证数据: ID=%d, Name=%s, Latitude=%f, Longitude=%f", city.ID, city.Name, city.Latitude, city.Longitude)

}

// TestFirstColInt64 测试FirstColInt64方法
func TestFirstColInt64(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况
	id, hasData, err := zmysql.FirstColInt64("SELECT id FROM cities WHERE id = ? LIMIT 1", 5)
	if err != nil {
		log.Fatalf("error querying city id: %v", err)
	}
	t.Logf("有数据测试 - id: %d, hasData: %t", id, hasData)

	// 测试无数据的情况
	id, hasData, err = zmysql.FirstColInt64("SELECT id FROM cities WHERE id = ? LIMIT 1", 999999)
	if err != nil {
		log.Fatalf("error querying non-existent city id: %v", err)
	}
	t.Logf("无数据测试 - id: %d, hasData: %t", id, hasData)

	// 测试NULL值的情况
	id, hasData, err = zmysql.FirstColInt64("SELECT NULL as id LIMIT 1")
	if err != nil {
		log.Fatalf("error querying null id: %v", err)
	}
	t.Logf("NULL值测试 - id: %d, hasData: %t", id, hasData)

	// 测试聚合函数
	count, hasData, err := zmysql.FirstColInt64("SELECT COUNT(*) FROM cities")
	if err != nil {
		log.Fatalf("error querying count: %v", err)
	}
	t.Logf("聚合函数测试 - count: %d, hasData: %t", count, hasData)
}

// TestFirstColString 测试FirstColString方法
func TestFirstColString(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况
	name, hasData, err := zmysql.FirstColString("SELECT name FROM cities WHERE id = ? LIMIT 1", 5)
	if err != nil {
		log.Fatalf("error querying city name: %v", err)
	}
	t.Logf("有数据测试 - name: %s, hasData: %t", name, hasData)

	// 测试无数据的情况
	name, hasData, err = zmysql.FirstColString("SELECT name FROM cities WHERE id = ? LIMIT 1", 999999)
	if err != nil {
		log.Fatalf("error querying non-existent city name: %v", err)
	}
	t.Logf("无数据测试 - name: %s, hasData: %t", name, hasData)

	// 测试NULL值的情况
	name, hasData, err = zmysql.FirstColString("SELECT NULL as name LIMIT 1")
	if err != nil {
		log.Fatalf("error querying null name: %v", err)
	}
	t.Logf("NULL值测试 - name: %s, hasData: %t", name, hasData)

	// 测试空字符串
	name, hasData, err = zmysql.FirstColString("SELECT '' as name LIMIT 1")
	if err != nil {
		log.Fatalf("error querying empty string: %v", err)
	}
	t.Logf("空字符串测试 - name: '%s', hasData: %t", name, hasData)
}

// TestFirstColProcInt64 测试FirstColProcInt64方法
func TestFirstColProcInt64(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况 - 假设存在名为 Proc_GetCityId 的存储过程
	id, hasData, err := zmysql.FirstColProcInt64("Proc_GetCityId", 6)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程有数据测试 - id: %d, hasData: %t", id, hasData)
	}

	// 测试无数据的情况
	id, hasData, err = zmysql.FirstColProcInt64("Proc_GetCityId", 999999)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程无数据测试 - id: %d, hasData: %t", id, hasData)
	}

	// 测试NULL值的情况 - 假设存在返回NULL的存储过程
	id, hasData, err = zmysql.FirstColProcInt64("Proc_GetNull")
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程NULL值测试 - id: %d, hasData: %t", id, hasData)
	}
}

// TestFirstColProcString 测试FirstColProcString方法
func TestFirstColProcString(t *testing.T) {
	// 创建 MySQL 客户端
	err := zmysql.Conn("root", "123456", "127.0.0.1:3326", "weather", zmysql.WithDebug())
	if err != nil {
		log.Fatalf("failed to create MySQL client: %v", err)
	}
	defer zmysql.Close()

	// 测试有数据的情况 - 假设存在名为 Proc_GetCityName 的存储过程
	name, hasData, err := zmysql.FirstColProcString("Proc_GetCityName", 5)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程有数据测试 - name: %s, hasData: %t", name, hasData)
	}

	// 测试无数据的情况
	name, hasData, err = zmysql.FirstColProcString("Proc_GetCityName", 999999)
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程无数据测试 - name: %s, hasData: %t", name, hasData)
	}

	// 测试NULL值的情况 - 假设存在返回NULL的存储过程
	name, hasData, err = zmysql.FirstColProcString("Proc_GetNullString")
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程NULL值测试 - name: %s, hasData: %t", name, hasData)
	}

	// 测试空字符串 - 假设存在返回空字符串的存储过程
	name, hasData, err = zmysql.FirstColProcString("Proc_GetEmptyString")
	if err != nil {
		t.Logf("存储过程不存在或执行失败: %v", err)
	} else {
		t.Logf("存储过程空字符串测试 - name: '%s', hasData: %t", name, hasData)
	}
}
