# Simple Workflow Examples

这个目录包含 simple-workflow 的使用示例。

## 重要说明

**这个 examples 目录是一个独立的 Go 模块**，它有自己的 `go.mod` 文件。这意味着：

- ✅ 你可以在示例中引入任何第三方依赖，**不会影响主项目的依赖**
- ✅ 示例代码的依赖不会被传递给使用主包的用户
- ✅ 可以自由添加演示、测试工具等，不用担心依赖污染

## 目录结构

```
examples/
├── go.mod           # 独立的模块定义
├── go.sum           # 独立的依赖锁定
├── README.md        # 本文件
├── QUICKSTART.md    # 快速开始指南
├── VERIFY.md        # 依赖独立性验证
├── with-sqlite/     # SQLite 完整示例
│   ├── main.go
│   ├── workflow_test.go
│   └── README.md
└── with-gin/        # Gin Web 框架示例
    ├── main.go
    └── README.md
```

## 运行示例

### 方式 1: 直接运行

```bash
# 运行基础示例
cd examples/basic
go run main.go

# 运行其他示例
cd examples/your-example
go run main.go
```

### 方式 2: 构建后运行

```bash
# 构建示例
cd examples/basic
go build -o basic

# 运行
./basic
```

## 添加新示例

### 1. 创建新目录

```bash
cd examples
mkdir my-example
cd my-example
```

### 2. 编写示例代码

```go
// examples/my-example/main.go
package main

import (
    "fmt"
    
    // 引入主包
    "github.com/blingmoon/simple-workflow/workflow"
    
    // 可以自由引入任何第三方包，不会影响主项目
    "github.com/gin-gonic/gin"  // 示例：引入 gin
)

func main() {
    // 使用 workflow
    service := workflow.NewWorkflowService(nil, nil)
    fmt.Println("Service created:", service)
    
    // 使用其他依赖
    r := gin.Default()
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })
}
```

### 3. 更新依赖

```bash
# 在 examples 目录下运行
cd /path/to/examples
go mod tidy
```

就这么简单！新的依赖只会添加到 `examples/go.mod`，不会影响主项目的 `go.mod`。

## 验证独立性

### 查看主项目依赖

```bash
cd /path/to/simple-workflow
go list -m all
```

### 查看示例项目依赖

```bash
cd /path/to/simple-workflow/examples
go list -m all
```

你会发现两者的依赖列表是不同的。

## 最佳实践

1. **保持示例简洁**：每个示例专注于演示一个特定功能
2. **添加注释**：充分注释代码，帮助用户理解
3. **独立运行**：确保每个示例可以独立运行
4. **文档完善**：在示例代码中说明使用场景和注意事项

## 示例列表

### 📦 with-sqlite/ - SQLite 完整示例

**推荐作为第一个学习示例！**

展示如何使用 SQLite 作为存储后端，包含完整的工作流创建、注册、执行流程。

**特性**：
- ✅ 使用 SQLite + GORM 进行数据持久化
- ✅ 完整的工作流生命周期演示
- ✅ 包含同步和异步任务节点
- ✅ JSONContext 使用示例
- ✅ 完整的单元测试

**快速开始**：
```bash
cd examples/with-sqlite
go run main.go
```

详见：[with-sqlite/README.md](./with-sqlite/README.md)

---

### 🌐 with-gin/ - Gin Web 框架集成

展示如何将 simple-workflow 集成到 Gin Web 应用中。

**特性**：
- ✅ HTTP API 触发工作流
- ✅ 查询工作流状态接口
- ✅ RESTful API 设计

**快速开始**：
```bash
cd examples/with-gin
go run main.go
```

详见：[with-gin/README.md](./with-gin/README.md)

## 常见问题

### Q: 为什么 examples 有独立的 go.mod？

A: 这样可以避免示例代码的依赖污染主项目。用户导入主包时，不会被迫下载示例代码的依赖。

### Q: 如何引用本地的主模块？

A: 在 `examples/go.mod` 中使用 `replace` 指令：

```go
replace github.com/blingmoon/simple-workflow => ../
```

这样在开发时会使用本地的代码，而不是远程版本。

### Q: 发布时需要注意什么？

A: 发布时确保：
1. `examples/go.mod` 中的 `replace` 指令保持不变（这是正常的）
2. 用户克隆仓库后，在 examples 目录运行 `go mod tidy` 即可

### Q: 可以在 examples 中使用哪些依赖？

A: 任何依赖都可以！常见的包括：
- Web 框架 (gin, echo, fiber)
- 数据库驱动 (gorm, sqlx)
- 配置管理 (viper, godotenv)
- 日志库 (zap, logrus)
- 等等...

完全不用担心会影响主项目！

