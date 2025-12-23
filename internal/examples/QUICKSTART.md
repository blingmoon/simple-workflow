# Examples 快速开始

## 🎯 核心优势

**examples 目录是一个独立的 Go 模块**，这意味着：

```
✅ 可以自由添加任何依赖（gin, echo, viper, zap 等）
✅ 不会污染主项目的 go.mod
✅ 用户导入主包时不会下载示例依赖
✅ 开发和演示更灵活
```

## 🚀 快速添加新示例

### 1. 创建示例目录

```bash
cd examples
mkdir my-example
cd my-example
```

### 2. 编写示例代码

创建 `main.go`：

```go
package main

import (
    "fmt"
    "github.com/blingmoon/simple-workflow/workflow"
    
    // 🎉 可以自由引入任何包，不会影响主项目！
    "github.com/gin-gonic/gin"
    "github.com/spf13/viper"
    "go.uber.org/zap"
)

func main() {
    // 使用 workflow
    service := workflow.NewWorkflowService(nil, nil)
    fmt.Println("Service created:", service)
    
    // 使用其他依赖演示功能
    // ...
}
```

### 3. 更新依赖

```bash
cd /path/to/simple-workflow/examples
go get github.com/gin-gonic/gin      # 添加你需要的依赖
go get github.com/spf13/viper
go mod tidy
```

### 4. 运行示例

```bash
cd /path/to/simple-workflow/examples/my-example
go run main.go
```

就这么简单！✨

## 📁 当前示例列表

### basic - 基础示例

最简单的示例，展示如何引用 workflow 包。

```bash
cd examples/basic
go run main.go
```

### with-gin - Web 服务示例

展示如何在示例中使用第三方依赖（gin 框架），构建 RESTful API。

```bash
cd examples/with-gin
go run main.go

# 访问 http://localhost:8080/ping 测试
```

## 🔍 验证依赖独立性

### 快速验证

```bash
# 主项目没有 gin
cd /path/to/simple-workflow
go list -m all | grep gin
# 输出：(无)

# 示例项目有 gin
cd /path/to/simple-workflow/examples  
go list -m all | grep gin
# 输出：github.com/gin-gonic/gin v1.11.0
```

更多验证方法请查看 [VERIFY.md](./VERIFY.md)

## 💡 常见示例模式

### 模式 1: Web API 示例

```go
// examples/web-api/main.go
package main

import (
    "github.com/blingmoon/simple-workflow/workflow"
    "github.com/gin-gonic/gin"
)

func main() {
    service := workflow.NewWorkflowService(repo, lock)
    
    r := gin.Default()
    r.POST("/workflow/create", func(c *gin.Context) {
        // 创建工作流示例
    })
    r.Run(":8080")
}
```

### 模式 2: CLI 工具示例

```go
// examples/cli-tool/main.go
package main

import (
    "github.com/blingmoon/simple-workflow/workflow"
    "github.com/spf13/cobra"
)

func main() {
    var rootCmd = &cobra.Command{
        Use:   "workflow-cli",
        Short: "Workflow command line tool",
    }
    
    rootCmd.AddCommand(createCmd)
    rootCmd.Execute()
}
```

### 模式 3: 配置管理示例

```go
// examples/with-config/main.go
package main

import (
    "github.com/blingmoon/simple-workflow/workflow"
    "github.com/spf13/viper"
)

func main() {
    // 加载配置
    viper.SetConfigFile("config.yaml")
    viper.ReadInConfig()
    
    // 使用配置创建服务
    service := workflow.NewWorkflowService(repo, lock)
}
```

### 模式 4: 完整应用示例

```go
// examples/full-app/main.go
package main

import (
    "github.com/blingmoon/simple-workflow/workflow"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "go.uber.org/zap"
)

func main() {
    // 初始化日志
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // 初始化数据库
    db, _ := gorm.Open(...)
    
    // 创建 workflow 服务
    repo := workflow.NewGormStore(db)
    lock := workflow.NewRedisLock(...)
    service := workflow.NewWorkflowService(repo, lock)
    
    // 启动 Web 服务
    r := gin.Default()
    // ... 设置路由
    r.Run(":8080")
}
```

## 📦 推荐的示例依赖

可以在 examples 中使用的常见包（不会影响主项目）：

### Web 框架
- `github.com/gin-gonic/gin` - HTTP web 框架
- `github.com/labstack/echo` - 高性能 web 框架
- `github.com/gofiber/fiber` - Express 风格框架

### 数据库
- `gorm.io/driver/mysql` - MySQL 驱动
- `gorm.io/driver/postgres` - PostgreSQL 驱动
- `github.com/go-redis/redis` - Redis 客户端

### 配置管理
- `github.com/spf13/viper` - 配置管理
- `github.com/joho/godotenv` - .env 文件支持

### 日志
- `go.uber.org/zap` - 高性能日志
- `github.com/sirupsen/logrus` - 结构化日志

### CLI
- `github.com/spf13/cobra` - CLI 框架
- `github.com/urfave/cli` - 命令行工具

### 测试
- `github.com/stretchr/testify` - 测试工具
- `github.com/golang/mock` - Mock 框架

## ❓ 常见问题

### Q: 为什么要独立的 go.mod？

A: 避免示例依赖污染主项目。用户导入主包时不会被迫下载示例依赖。

### Q: 如何添加新依赖？

A: 在 examples 目录下运行：

```bash
cd examples
go get github.com/your/package
go mod tidy
```

### Q: 发布后用户需要下载示例依赖吗？

A: 不需要！除非用户主动进入 examples 目录并运行 `go mod download`。

### Q: 可以有多个示例子目录吗？

A: 可以！所有示例共享同一个 `examples/go.mod`：

```
examples/
├── go.mod          # 共享
├── basic/
├── with-gin/
├── with-cli/
└── full-app/
```

### Q: 如何在示例中使用最新的主包代码？

A: `examples/go.mod` 中的 `replace` 指令会自动使用本地代码：

```go
replace github.com/blingmoon/simple-workflow => ../
```

修改主包后，示例会立即使用新代码，无需发布版本。

## 🎓 最佳实践

1. **每个示例专注一个功能点** - 不要在一个示例中演示太多东西
2. **添加详细注释** - 帮助用户理解代码意图
3. **提供 README** - 在示例目录下添加 README.md 说明
4. **保持可运行性** - 确保示例可以独立运行
5. **使用真实场景** - 示例应该贴近实际使用场景

## 🔗 相关文档

- [README.md](./README.md) - 完整说明文档
- [VERIFY.md](./VERIFY.md) - 验证依赖独立性
- [主项目 README](../README.md) - 项目主文档

## 🤝 贡献示例

欢迎提交新的示例！请确保：

1. 示例代码清晰易懂
2. 添加必要的注释
3. 在示例目录下添加 README.md
4. 运行 `go mod tidy` 更新依赖
5. 验证示例可以正常运行

---

**开始创建你的第一个示例吧！** 🚀

