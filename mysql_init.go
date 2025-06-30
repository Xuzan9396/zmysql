package zmysql

import (
	"github.com/Xuzan9396/zmysql/smysql"
	"time"
	//_ "github.com/go-sql-driver/mysql"
	// 引入 MySQL 驱动
)

var mysql_client *smysql.MySQLClient

// Conn 创建并初始化一个新的 MySQL 客户端
func Conn(username, password, addr, dbName string, opts ...func(*smysql.MySQLClient)) error {
	client, err := smysql.Conn(username, password, addr, dbName, opts...)
	if err != nil {
		return err
	}
	mysql_client = client
	return nil
}

// 设置连接最大生命周期
func WithConnMaxLifetime(d time.Duration) func(*smysql.MySQLClient) {
	return smysql.WithConnMaxLifetime(d)
}

// 设置最大连接数
func WithMaxOpenConns(n int) func(*smysql.MySQLClient) {
	return smysql.WithMaxOpenConns(n)
}

// 设置最大空闲连接数
func WithMaxIdleConns(n int) func(*smysql.MySQLClient) {
	return smysql.WithMaxIdleConns(n)
}

// 设置时区
func WithLoc(loc string) func(*smysql.MySQLClient) {
	return smysql.WithLoc(loc)
}

// WithDebug 启用调试模式，打印 SQL 语句和参数
func WithDebug() func(*smysql.MySQLClient) {
	return smysql.WithDebug()
}

// Close 关闭数据库连接
func Close() error {
	return mysql_client.Close()
}
