# ZMySQL - Go MySQL 数据库客户端库

ZMySQL 是一个简单易用的 Go MySQL 数据库客户端库，提供类似 ORM 的接口，支持两种使用模式：全局客户端和实例客户端。

## 特性

- 🚀 **双模式支持**：支持全局客户端和实例客户端两种使用方式
- 🎯 **类型安全**：支持泛型，提供类型安全的数据库操作
- 🔄 **自动映射**：使用 `db` 标签自动进行结构体到 SQL 字段映射
- 🛡️ **NULL 处理**：智能处理 NULL 值，自动转换为 Go 零值
- 📊 **多种查询**：支持单条、多条、数组、映射等多种查询模式
- 🔧 **连接池**：内置连接池管理，支持连接配置
- 🐛 **调试模式**：可选的 SQL 查询日志记录
- ⚡ **高性能**：使用预处理语句和连接池优化性能

## 安装

```bash
go get github.com/Xuzan9396/zmysql
```

## 快速开始

### 全局客户端模式

```go
package main

import (
    "fmt"
    "github.com/Xuzan9396/zmysql"
    "github.com/Xuzan9396/zmysql/smysql"
)

type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

func main() {
    // 初始化全局连接
    err := zmysql.Conn("username", "password", "localhost:3306", "database", 
        smysql.WithDebug())
    if err != nil {
        panic(err)
    }
    
    // 查询用户
    var users []User
    err = zmysql.Find(&users, "SELECT * FROM users WHERE age > ?", 18)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d users\n", len(users))
}
```

### 实例客户端模式

```go
package main

import (
    "fmt"
    "github.com/Xuzan9396/zmysql/smysql"
)

func main() {
    // 创建客户端实例
    client, err := smysql.Conn("username", "password", "localhost:3306", "database",
        smysql.WithDebug(),
        smysql.WithMaxOpenConns(100))
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // 使用实例进行查询
    var users []User
    err = client.Find(&users, "SELECT * FROM users WHERE age > ?", 18)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d users\n", len(users))
}
```

## 连接配置选项

### WithDebug() - 调试模式

启用 SQL 查询日志记录，便于开发和调试。

```go
// 全局模式
err := zmysql.Conn("user", "pass", "localhost:3306", "db", smysql.WithDebug())

// 实例模式
client, err := smysql.Conn("user", "pass", "localhost:3306", "db", smysql.WithDebug())
```

### WithMaxOpenConns() - 最大连接数

设置连接池的最大连接数。

```go
client, err := smysql.Conn("user", "pass", "localhost:3306", "db", 
    smysql.WithMaxOpenConns(100))
```

### WithMaxIdleConns() - 最大空闲连接数

设置连接池的最大空闲连接数。

```go
client, err := smysql.Conn("user", "pass", "localhost:3306", "db", 
    smysql.WithMaxIdleConns(50))
```

### WithConnMaxLifetime() - 连接最大生存时间

设置连接的最大生存时间。

```go
import "time"

client, err := smysql.Conn("user", "pass", "localhost:3306", "db", 
    smysql.WithConnMaxLifetime(4*time.Hour))
```

### WithLoc() - 时区设置

设置数据库连接的时区。

```go
client, err := smysql.Conn("user", "pass", "localhost:3306", "db", 
    smysql.WithLoc("UTC"))
```

## 基础查询功能

### Find() - 查询多条记录到结构体切片

查询多条记录并自动映射到结构体切片。

```go
type User struct {
    ID       int       `db:"id"`
    Name     string    `db:"name"`
    Email    string    `db:"email"`
    CreateAt time.Time `db:"created_at"`
}

// 查询所有用户
var users []User
err := zmysql.Find(&users, "SELECT * FROM users")

// 带条件查询
err = zmysql.Find(&users, "SELECT * FROM users WHERE age > ? AND status = ?", 18, "active")

// 实例模式
err = client.Find(&users, "SELECT * FROM users WHERE age > ?", 18)
```

### First() - 查询单条记录到结构体

查询单条记录，返回是否找到记录的布尔值。

```go
var user User

// 查询单个用户
found, err := zmysql.First(&user, "SELECT * FROM users WHERE id = ?", 1)
if err != nil {
    panic(err)
}

if found {
    fmt.Printf("Found user: %+v\n", user)
} else {
    fmt.Println("User not found")
}

// 实例模式
found, err = client.First(&user, "SELECT * FROM users WHERE email = ?", "user@example.com")
```

### FirstCol() - 查询单列值到基础类型

查询单列值并映射到基础类型。

```go
// 查询字符串
var name string
found, err := zmysql.FirstCol(&name, "SELECT name FROM users WHERE id = ?", 1)

// 查询整数
var count int64
found, err = zmysql.FirstCol(&count, "SELECT COUNT(*) FROM users")

// 查询浮点数
var avgAge float64
found, err = zmysql.FirstCol(&avgAge, "SELECT AVG(age) FROM users")

// 查询布尔值
var isActive bool
found, err = zmysql.FirstCol(&isActive, "SELECT is_active FROM users WHERE id = ?", 1)

// 实例模式
found, err = client.FirstCol(&name, "SELECT name FROM users WHERE id = ?", 1)
```

## 类型化单列查询

### FirstColInt64() - 查询int64类型单列

专门用于查询 int64 类型的单列值，返回值、是否找到、错误。

```go
// 查询用户数量
count, found, err := zmysql.FirstColInt64("SELECT COUNT(*) FROM users")
if err != nil {
    panic(err)
}
if found {
    fmt.Printf("Total users: %d\n", count)
}

// 查询用户ID
userID, found, err := zmysql.FirstColInt64("SELECT id FROM users WHERE email = ?", "user@example.com")

// 查询最大年龄
maxAge, found, err := zmysql.FirstColInt64("SELECT MAX(age) FROM users")

// 实例模式
count, found, err = client.FirstColInt64("SELECT COUNT(*) FROM users WHERE active = ?", true)
```

### FirstColString() - 查询string类型单列

专门用于查询 string 类型的单列值。

```go
// 查询用户名
name, found, err := zmysql.FirstColString("SELECT name FROM users WHERE id = ?", 1)
if err != nil {
    panic(err)
}
if found {
    fmt.Printf("User name: %s\n", name)
}

// 查询邮箱
email, found, err := zmysql.FirstColString("SELECT email FROM users WHERE id = ?", 1)

// 查询状态
status, found, err := zmysql.FirstColString("SELECT status FROM users WHERE id = ?", 1)

// 实例模式
name, found, err = client.FirstColString("SELECT name FROM users WHERE id = ?", 1)
```

## 数组查询功能

### FindArrayInt64() - 查询指定字段的int64数组

查询指定字段并返回 int64 类型的数组，自动过滤 NULL 值。

```go
// 查询所有用户ID
userIDs, err := zmysql.FindArrayInt64("id", "SELECT id, name FROM users WHERE active = ?", true)
if err != nil {
    panic(err)
}
fmt.Printf("Active user IDs: %v\n", userIDs)

// 查询特定年龄段的用户ID
ageGroupIDs, err := zmysql.FindArrayInt64("id", "SELECT id FROM users WHERE age BETWEEN ? AND ?", 18, 65)

// 查询分数
scores, err := zmysql.FindArrayInt64("score", "SELECT score FROM user_scores WHERE user_id = ?", 1)

// 实例模式
userIDs, err = client.FindArrayInt64("id", "SELECT id FROM users WHERE department_id = ?", 1)
```

### FindArrayString() - 查询指定字段的string数组

查询指定字段并返回 string 类型的数组，自动过滤 NULL 值。

```go
// 查询所有用户名
userNames, err := zmysql.FindArrayString("name", "SELECT name, email FROM users WHERE active = ?", true)
if err != nil {
    panic(err)
}
fmt.Printf("Active user names: %v\n", userNames)

// 查询邮箱列表
emails, err := zmysql.FindArrayString("email", "SELECT email FROM users WHERE department = ?", "IT")

// 查询城市列表
cities, err := zmysql.FindArrayString("city", "SELECT DISTINCT city FROM users")

// 实例模式
userNames, err = client.FindArrayString("name", "SELECT name FROM users WHERE role = ?", "admin")
```

## 执行功能

### Exec() - 执行INSERT/UPDATE/DELETE操作

执行数据修改操作，返回是否影响了行。

```go
// 插入用户
success, err := zmysql.Exec("INSERT INTO users (name, email, age) VALUES (?, ?, ?)", 
    "John Doe", "john@example.com", 25)
if err != nil {
    panic(err)
}
if success {
    fmt.Println("User inserted successfully")
}

// 更新用户
success, err = zmysql.Exec("UPDATE users SET age = ? WHERE id = ?", 26, 1)

// 删除用户
success, err = zmysql.Exec("DELETE FROM users WHERE id = ?", 1)

// 批量更新
success, err = zmysql.Exec("UPDATE users SET status = ? WHERE department = ?", "inactive", "old_dept")

// 实例模式
success, err = client.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Jane", "jane@example.com")
```

### ExecFindLastId() - 获取最后插入的ID

执行 INSERT 操作并返回最后插入的自增ID。

```go
// 插入用户并获取ID
lastID, err := zmysql.ExecFindLastId("INSERT INTO users (name, email, age) VALUES (?, ?, ?)", 
    "Alice Smith", "alice@example.com", 30)
if err != nil {
    panic(err)
}
fmt.Printf("New user ID: %d\n", lastID)

// 插入订单并获取订单ID
orderID, err := zmysql.ExecFindLastId("INSERT INTO orders (user_id, amount, status) VALUES (?, ?, ?)", 
    1, 99.99, "pending")

// 实例模式
lastID, err = client.ExecFindLastId("INSERT INTO products (name, price) VALUES (?, ?)", 
    "New Product", 29.99)
```

### ExecByte() - 执行查询并返回JSON字节数据

执行查询并返回原始 JSON 格式的字节数据，适用于需要返回动态结构数据的场景。

```go
import "github.com/Xuzan9396/zmysql/smysql"

// 查询单条记录
data, err := zmysql.ExecByte("SELECT id, name, email FROM users WHERE id = ?", 
    smysql.HAS_ONE, 1)
if err != nil {
    panic(err)
}
fmt.Printf("User data: %s\n", string(data))
// 输出: {"id":1,"name":"John","email":"john@example.com"}

// 查询多条记录
data, err = zmysql.ExecByte("SELECT id, name FROM users WHERE active = ?", 
    smysql.HAS_LIST, true)
fmt.Printf("Users data: %s\n", string(data))
// 输出: [{"id":1,"name":"John"},{"id":2,"name":"Jane"}]

// 实例模式
data, err = client.ExecByte("SELECT * FROM products WHERE category = ?", 
    smysql.HAS_LIST, "electronics")
```

## 泛型功能（包级函数）

### FindArray[T] - 泛型数组查询

使用泛型查询指定字段的数组，支持 int64 和 string 类型。

```go
import "github.com/Xuzan9396/zmysql/smysql"

// 查询 int64 数组
userIDs, err := smysql.FindArray[int64](client, "id", "SELECT id FROM users WHERE active = ?", true)
if err != nil {
    panic(err)
}
fmt.Printf("User IDs: %v\n", userIDs)

// 查询 string 数组
userNames, err := smysql.FindArray[string](client, "name", "SELECT name FROM users WHERE department = ?", "IT")
fmt.Printf("IT Users: %v\n", userNames)

// 注意：泛型函数需要传入客户端实例
// 全局模式需要先获取实例：
// client := zmysql.GetClient() // 如果提供了此方法
```

### FirstColAny[T] - 泛型单列查询

使用泛型查询单列值，支持 int64 和 string 类型。

```go
// 查询 int64 类型
count, found, err := smysql.FirstColAny[int64](client, "SELECT COUNT(*) FROM users")
if err != nil {
    panic(err)
}
if found {
    fmt.Printf("Total count: %d\n", count)
}

// 查询 string 类型
name, found, err := smysql.FirstColAny[string](client, "SELECT name FROM users WHERE id = ?", 1)
if found {
    fmt.Printf("User name: %s\n", name)
}

// 查询最大值
maxAge, found, err := smysql.FirstColAny[int64](client, "SELECT MAX(age) FROM users")
```

### FindMap[T,Y] - 泛型映射查询

使用泛型查询并返回键值对映射，支持多种类型组合。

```go
// map[int64]string - ID到姓名的映射
userMap, err := smysql.FindMap[int64, string](client, "id", "name", 
    "SELECT id, name FROM users WHERE active = ?", true)
if err != nil {
    panic(err)
}
fmt.Printf("User mapping: %v\n", userMap)
// 输出: map[1:John 2:Jane 3:Bob]

// map[string]int64 - 姓名到ID的映射
nameToID, err := smysql.FindMap[string, int64](client, "name", "id", 
    "SELECT name, id FROM users")

// map[int64]User - ID到完整用户结构体的映射（valueField为空时返回完整结构体）
type User struct {
    ID   int64  `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

userStructMap, err := smysql.FindMap[int64, User](client, "id", "", 
    "SELECT * FROM users WHERE department = ?", "IT")
```

## 存储过程支持

所有查询方法都有对应的存储过程版本：

```go
// 存储过程查询多条记录
var users []User
err := zmysql.FindProc(&users, "GetActiveUsers", 18)

// 存储过程查询单条记录
var user User
found, err := zmysql.FirstProc(&user, "GetUserById", 1)

// 存储过程单列查询
count, found, err := zmysql.FirstColProc(&count, "GetUserCount", "active")

// 存储过程数组查询
userIDs, err := zmysql.FindProcArrayInt64("id", "GetUserIdsByDept", "IT")
userNames, err := zmysql.FindProcArrayString("name", "GetUserNamesByRole", "admin")

// 存储过程字节查询
data, err := zmysql.ExecProcByte("GetUserStats", smysql.HAS_LIST, "2023")

// 存储过程泛型查询
ids, err := smysql.FindProcArray[int64](client, "id", "GetActiveUserIds")
userMap, err := smysql.FindProcMap[int64, string](client, "id", "name", "GetUserMapping")
```

## 多结果集查询

处理返回多个结果集的存储过程：

```go
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
}

type Stats struct {
    Total int `db:"total"`
}

type Summary struct {
    Summary string `db:"summary"`
}

// 定义多个目标
var users []User
var stats Stats
var summary Summary

// 调用返回多个结果集的存储过程
err := zmysql.FindMultipleProc([]any{&users, &stats, &summary}, "GetCompleteReport", 2023)
if err != nil {
    panic(err)
}

fmt.Printf("Users: %v\n", users)
fmt.Printf("Stats: %v\n", stats)
fmt.Printf("Summary: %v\n", summary)
```

## 错误处理

```go
// 基本错误处理
var users []User
err := zmysql.Find(&users, "SELECT * FROM users")
if err != nil {
    // 处理错误
    log.Printf("Query failed: %v", err)
    return
}

// 检查是否找到记录
var user User
found, err := zmysql.First(&user, "SELECT * FROM users WHERE id = ?", 999)
if err != nil {
    log.Printf("Query error: %v", err)
    return
}

if !found {
    log.Println("User not found")
    return
}

// 执行操作结果检查
success, err := zmysql.Exec("UPDATE users SET status = ? WHERE id = ?", "inactive", 1)
if err != nil {
    log.Printf("Update failed: %v", err)
    return
}

if !success {
    log.Println("No rows affected")
}
```

## 结构体映射

使用 `db` 标签进行字段映射：

```go
type User struct {
    ID          int64     `db:"id"`
    FirstName   string    `db:"first_name"`    // 映射到 first_name 列
    LastName    string    `db:"last_name"`     // 映射到 last_name 列
    Email       string    `db:"email"`
    Age         int       `db:"age"`
    IsActive    bool      `db:"is_active"`     // 映射到 is_active 列
    CreatedAt   time.Time `db:"created_at"`    // 映射到 created_at 列
    UpdatedAt   time.Time `db:"updated_at"`    // 映射到 updated_at 列
    ProfilePic  []byte    `db:"profile_pic"`   // 二进制数据
    Score       float64   `db:"score"`         // 浮点数
    UnmappedField string                       // 没有 db 标签的字段会被忽略
}
```

## NULL 值处理

ZMySQL 自动处理 NULL 值：

```go
type UserProfile struct {
    ID          int64     `db:"id"`
    Name        string    `db:"name"`
    Phone       string    `db:"phone"`        // NULL 值会设置为空字符串 ""
    Age         int       `db:"age"`          // NULL 值会设置为 0
    IsVerified  bool      `db:"is_verified"`  // NULL 值会设置为 false
    LastLogin   time.Time `db:"last_login"`   // NULL 值会设置为零时间
    Score       float64   `db:"score"`        // NULL 值会设置为 0.0
}

// 查询可能包含 NULL 值的数据
var profiles []UserProfile
err := zmysql.Find(&profiles, "SELECT * FROM user_profiles")
// NULL 值会被自动转换为对应类型的零值
```

## 事务处理

虽然 ZMySQL 主要专注于简单查询，但您可以通过获取底层数据库连接来处理事务：

```go
// 实例模式下访问底层连接
client, err := smysql.Conn("user", "pass", "localhost:3306", "db")
if err != nil {
    panic(err)
}

// 开始事务
tx, err := client.DB.Begin()
if err != nil {
    panic(err)
}

// 在事务中执行操作
_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", "John")
if err != nil {
    tx.Rollback()
    panic(err)
}

_, err = tx.Exec("UPDATE accounts SET balance = balance - 100 WHERE user_id = ?", 1)
if err != nil {
    tx.Rollback()
    panic(err)
}

// 提交事务
err = tx.Commit()
if err != nil {
    panic(err)
}
```

## 性能优化建议

1. **使用连接池**：
```go
client, err := smysql.Conn("user", "pass", "localhost:3306", "db",
    smysql.WithMaxOpenConns(100),        // 最大连接数
    smysql.WithMaxIdleConns(50),         // 最大空闲连接数
    smysql.WithConnMaxLifetime(4*time.Hour)) // 连接最大生存时间
```

2. **使用预处理语句**：
```go
// ZMySQL 内部自动使用预处理语句，无需额外配置
var users []User
err := zmysql.Find(&users, "SELECT * FROM users WHERE age > ?", 18)
```

3. **批量操作**：
```go
// 批量插入时可以使用事务
tx, _ := client.DB.Begin()
for _, user := range users {
    tx.Exec("INSERT INTO users (name, email) VALUES (?, ?)", user.Name, user.Email)
}
tx.Commit()
```

## 调试和日志

启用调试模式查看执行的 SQL 语句：

```go
// 启用调试模式
client, err := smysql.Conn("user", "pass", "localhost:3306", "db", 
    smysql.WithDebug())

// 执行查询时会输出 SQL 语句和参数
var users []User
err = client.Find(&users, "SELECT * FROM users WHERE age > ?", 18)
// 日志输出: sql:SELECT * FROM users WHERE age > ?, args:[18]
```

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/Xuzan9396/zmysql"
    "github.com/Xuzan9396/zmysql/smysql"
)

type User struct {
    ID        int64     `db:"id"`
    Name      string    `db:"name"`
    Email     string    `db:"email"`
    Age       int       `db:"age"`
    IsActive  bool      `db:"is_active"`
    CreatedAt time.Time `db:"created_at"`
}

func main() {
    // 初始化连接
    err := zmysql.Conn("root", "password", "localhost:3306", "testdb",
        smysql.WithDebug(),
        smysql.WithMaxOpenConns(50))
    if err != nil {
        log.Fatal(err)
    }

    // 1. 查询所有用户
    var users []User
    err = zmysql.Find(&users, "SELECT * FROM users WHERE is_active = ?", true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d active users\n", len(users))

    // 2. 查询单个用户
    var user User
    found, err := zmysql.First(&user, "SELECT * FROM users WHERE id = ?", 1)
    if err != nil {
        log.Fatal(err)
    }
    if found {
        fmt.Printf("User: %+v\n", user)
    }

    // 3. 查询用户数量
    count, found, err := zmysql.FirstColInt64("SELECT COUNT(*) FROM users")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", count)

    // 4. 查询用户ID数组
    userIDs, err := zmysql.FindArrayInt64("id", "SELECT id FROM users WHERE age > ?", 18)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Adult user IDs: %v\n", userIDs)

    // 5. 插入新用户
    lastID, err := zmysql.ExecFindLastId("INSERT INTO users (name, email, age, is_active) VALUES (?, ?, ?, ?)",
        "New User", "new@example.com", 25, true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("New user created with ID: %d\n", lastID)

    // 6. 更新用户
    success, err := zmysql.Exec("UPDATE users SET age = ? WHERE id = ?", 26, lastID)
    if err != nil {
        log.Fatal(err)
    }
    if success {
        fmt.Println("User updated successfully")
    }

    // 7. 使用实例客户端的泛型功能
    client, err := smysql.Conn("root", "password", "localhost:3306", "testdb")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 泛型数组查询
    names, err := smysql.FindArray[string](client, "name", "SELECT name FROM users WHERE is_active = ?", true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Active user names: %v\n", names)

    // 泛型映射查询
    userMap, err := smysql.FindMap[int64, string](client, "id", "name", "SELECT id, name FROM users")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User ID to Name mapping: %v\n", userMap)
}
```

## 注意事项

1. **结构体字段必须导出**：结构体字段必须以大写字母开头才能被正确映射。

2. **db 标签必须匹配列名**：确保 `db` 标签中的名称与数据库列名完全匹配。

3. **NULL 值处理**：数据库中的 NULL 值会被转换为 Go 类型的零值。

4. **泛型约束**：泛型函数目前支持 `int64` 和 `string` 类型，以及 `comparable` 类型作为映射的键。

5. **连接管理**：实例模式下记得调用 `client.Close()` 关闭连接。

6. **SQL 注入防护**：始终使用参数化查询，避免直接拼接 SQL 字符串。

## 许可证

MIT License