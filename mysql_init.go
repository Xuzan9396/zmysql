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
	//_ "github.com/go-sql-driver/mysql"
	// 引入 MySQL 驱动
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

// 获取结构体字段映射信息
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

// Close 关闭数据库连接
func Close() error {
	return mysql_client.DB.Close()
}
