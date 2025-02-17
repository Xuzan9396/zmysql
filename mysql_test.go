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
	err = zmysql.Find(&cities, "SELECT  name as city_name, latitude, longitude,created_at FROM cities WHERE id = ? limit 5", 1)
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
		Name       string  `db:"city_name"`
		ID         int     `db:"id"`
		Latitude   float64 `db:"latitude"`
		Longitude  float64 `db:"longitude"`
		WikiDataId string  `db:"wikiDataId"`
	}
	// 查询数据
	var cities CityInfo
	bools, err := zmysql.First(&cities, "SELECT  name as city_name, latitude, longitude FROM cities WHERE id = ? limit 1", 1)
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

	t.Log("长度", len(cities))

	for i, city := range cities {
		t.Log(i, city.ID, city.Name, city.WikiDataId == "")
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

	bools, err := zmysql.Exec("UPDATE cities SET created_at = ?,wikiDataId = ? WHERE id = ?", "2025-02-13 11:49:36", nil, 1)
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
	bools, err := zmysql.FirstCol(&cityName, "SELECT  name as city_name FROM cities WHERE id = ? limit 1", 1)
	if err != nil {
		log.Fatalf("error querying cities: %v", err)
	}
	if bools != true {
		t.Log("没有数据")
		return
	}
	t.Log(cityName)
}
