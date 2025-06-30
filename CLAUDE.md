# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在此代码库中工作时提供指导。

## 编码规范
1. 需要每个方法添加注释，描述其功能和参数

## 项目概述

这是 `zmysql`，一个 Go MySQL 数据库客户端库，为 MySQL 操作提供类似 ORM 的接口。该库专注于简单性和易用性，提供两种不同的 API 模式：

1. **全局客户端模式**（主包）：使用通过全局函数访问的单例客户端
2. **实例客户端模式**（smysql 子包）：返回客户端实例以实现更明确的控制

## 架构

### 核心组件

- **连接管理**：数据库连接处理，具有可配置的连接池
- **查询映射**：使用 `db` 结构体标签自动进行结构体到 SQL 字段映射
- **基于反射的 ORM**：大量使用 Go 反射进行动态结构体扫描和映射
- **调试日志**：通过 zlog 可选的 SQL 查询和参数日志记录

### 关键文件

- `mysql_init.go`：连接初始化和配置选项
- `mysql_select.go`：查询操作（Find、First、存储过程）
- `mysql_exec.go`：执行操作和原始 JSON 结果处理
- `constvars.go`：常量和类型定义
- `smysql/mysql.go`：基于实例的客户端替代实现

## 常用命令

### 测试
```bash
go test -v                    # 在当前目录运行测试
go test -v ./smysql          # 在 smysql 子目录运行测试
```

### 构建和依赖管理
```bash
go mod tidy                  # 清理依赖
go build                     # 构建库
```

## 关键使用模式

### 连接设置
- 全局客户端：`zmysql.Conn(username, password, addr, dbName, options...)`
- 实例客户端：`client, err := smysql.Conn(username, password, addr, dbName, options...)`

### 配置选项
- `WithDebug()`：启用 SQL 查询日志记录
- `WithLoc(timezone)`：设置数据库时区
- `WithMaxOpenConns(n)`：配置连接池
- `WithConnMaxLifetime(duration)`：设置连接生存时间

### 查询操作
- `Find()`：查询多条记录到切片
- `First()`：查询单条记录到结构体
- `FirstCol()`：查询单列值
- `*Proc()`：执行存储过程
- `Exec()`：执行 INSERT/UPDATE/DELETE 操作

### 字段映射
- 使用 `db` 结构体标签进行列映射
- 自动 NULL 值处理，使用适当的 Go 零值
- 基于反射的字段赋值，具有类型安全性

## 开发注意事项

- 代码库在某些地方使用中文注释和错误消息
- 存在两个并行实现（主包 vs smysql），API 略有差异
- 大量依赖反射实现动态行为
- 使用连接池和预处理语句提高性能
- 调试模式通过 zlog 打印 SQL 查询和参数

