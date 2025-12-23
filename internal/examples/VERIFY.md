# 验证 Examples 依赖独立性

这个文档展示如何验证 examples 目录的依赖不会污染主项目。

## 快速验证

### 1. 查看主项目依赖

```bash
cd /path/to/simple-workflow
go list -m all | grep gin
```

**预期结果**：没有输出（主项目不依赖 gin）

### 2. 查看示例项目依赖

```bash
cd /path/to/simple-workflow/examples
go list -m all | grep gin
```

**预期结果**：显示 gin 相关依赖

```
github.com/gin-contrib/sse v1.1.0
github.com/gin-gonic/gin v1.11.0
```

## 详细验证步骤

### 步骤 1: 检查主项目 go.mod

```bash
cd /path/to/simple-workflow
cat go.mod
```

应该看到类似这样的依赖（**没有 gin**）：

```go
module github.com/blingmoon/simple-workflow

go 1.24.0

require (
	github.com/go-playground/validator/v10 v10.30.0
	github.com/pkg/errors v0.9.1
	gorm.io/gorm v1.31.1
)
```

### 步骤 2: 检查示例项目 go.mod

```bash
cd /path/to/simple-workflow/examples
cat go.mod
```

应该看到 gin 依赖：

```go
module github.com/blingmoon/simple-workflow/examples

go 1.24.0

replace github.com/blingmoon/simple-workflow => ../

require (
	github.com/blingmoon/simple-workflow v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.11.0  // ← 示例项目特有的依赖
)
```

### 步骤 3: 运行示例验证

```bash
# 运行基础示例（不需要 gin）
cd /path/to/simple-workflow/examples/basic
go run main.go

# 运行 gin 示例（需要 gin）
cd /path/to/simple-workflow/examples/with-gin
go run main.go
```

### 步骤 4: 验证构建主项目

```bash
cd /path/to/simple-workflow
go build ./...
```

**预期结果**：成功构建，不会下载 gin 相关依赖

## 工作原理

### 独立模块

```
simple-workflow/
├── go.mod              # 主模块
├── workflow/
│   └── *.go
└── examples/
    ├── go.mod          # 独立的子模块（关键！）
    ├── basic/
    │   └── main.go
    └── with-gin/
        └── main.go
```

### 依赖关系

```
主模块 (simple-workflow)
  ├── github.com/pkg/errors
  ├── gorm.io/gorm
  └── ...

示例模块 (simple-workflow/examples) - 独立！
  ├── github.com/blingmoon/simple-workflow (通过 replace 指向本地)
  ├── github.com/gin-gonic/gin (示例特有)
  └── ... (其他示例依赖)
```

### 用户导入主包时

当用户在他们的项目中导入：

```go
import "github.com/blingmoon/simple-workflow/workflow"
```

Go 模块系统会：
1. 下载主模块 `simple-workflow`
2. 解析主模块的 `go.mod`
3. **不会**解析 `examples/go.mod`（因为它是独立模块）
4. **不会**下载 gin 等示例依赖

## 常见问题

### Q: 为什么需要 replace 指令？

A: `replace` 指令让示例在开发时使用本地代码：

```go
replace github.com/blingmoon/simple-workflow => ../
```

这样修改主包代码后，示例会立即使用新代码，无需发布版本。

### Q: 发布到 GitHub 后用户会受影响吗？

A: 不会！用户克隆仓库后：

```bash
# 用户导入主包
import "github.com/blingmoon/simple-workflow/workflow"
# → 只会下载主模块依赖

# 如果用户想运行示例
cd examples
go mod download  # 单独下载示例依赖
go run basic/main.go
```

### Q: 可以添加多少依赖到 examples？

A: 随意添加！常见的包括：

- **Web 框架**: gin, echo, fiber, chi
- **数据库**: gorm, sqlx, mongo-driver
- **配置**: viper, godotenv
- **日志**: zap, logrus, zerolog
- **测试**: testify, gomock
- **工具**: cobra, urfave/cli

完全不用担心污染主项目！

## 验证命令汇总

```bash
# 主项目依赖
cd /path/to/simple-workflow
go list -m all

# 示例项目依赖
cd /path/to/simple-workflow/examples
go list -m all

# 对比差异
diff <(cd /path/to/simple-workflow && go list -m all | sort) \
     <(cd /path/to/simple-workflow/examples && go list -m all | sort)
```

## 结论

✅ **主项目 go.mod**: 干净，只包含核心依赖  
✅ **示例项目 go.mod**: 包含所有演示需要的依赖  
✅ **用户导入主包**: 不会被迫下载示例依赖  
✅ **开发体验**: 示例可以自由使用任何依赖

