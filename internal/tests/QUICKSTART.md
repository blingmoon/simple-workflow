# 快速开始 - Simple Workflow 测试

## 🚀 5分钟快速上手

### 1. 进入测试目录

```bash
cd tests
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行所有测试

```bash
go test -v ./...
```

### 4. 查看测试结果

```
PASS
ok  	github.com/blingmoon/simple-workflow/tests	2.520s
```

## 📝 测试文件说明

### workflow_basic_test.go

基础工作流功能测试，包含：

```go
// 测试工作流创建
func TestWorkflowCreationBasic(t *testing.T) {
    // 1. 定义工作流配置
    // 2. 加载配置
    // 3. 注册任务处理器
    // 4. 创建工作流实例
}

// 测试工作流执行
func TestWorkflowExecution(t *testing.T) {
    // 测试单任务和多任务工作流执行
}

// 测试工作流查询
func TestWorkflowQuery(t *testing.T) {
    // 查询工作流实例和统计数量
}
```

### json_context_simple_test.go

JSON 上下文测试：

```go
func TestJSONContextSimple(t *testing.T) {
    // 测试 JSON 上下文的创建、读取、设置、嵌套等操作
}
```

### integration_test.go

集成测试，包含完整场景：

```go
// 订单处理工作流
func TestCompleteWorkflowScenario(t *testing.T) {
    // 验证 -> 支付 -> 发货
}

// 异步工作流
func TestAsyncWorkflow(t *testing.T) {
    // 带异步检查的任务
}

// 错误处理
func TestErrorHandling(t *testing.T) {
    // 测试各种错误场景
}
```

## 🎯 常用命令

### 运行特定测试

```bash
# 只运行 JSON 上下文测试
go test -v -run TestJSONContextSimple

# 只运行工作流创建测试
go test -v -run TestWorkflowCreationBasic

# 只运行集成测试
go test -v -run TestCompleteWorkflowScenario
```

### 查看测试覆盖率

```bash
# 显示覆盖率
go test -v -cover ./...

# 生成覆盖率报告
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 在浏览器中查看
open coverage.html
```

### 运行性能测试

```bash
go test -v -bench=. -benchmem
```

### 并行运行测试

```bash
go test -v -parallel 4 ./...
```

## 📚 测试示例

### 示例 1: 创建简单工作流

```go
func TestSimpleWorkflow(t *testing.T) {
    // 1. 创建服务
    service := setupTestService(t)
    
    // 2. 定义工作流
    workflowConfigJSON := `{
        "id": "my_workflow",
        "name": "我的工作流",
        "nodes": [
            {"id": "task1", "name": "任务1", "next_nodes": []}
        ]
    }`
    
    var config workflow.WorkflowConfig
    json.Unmarshal([]byte(workflowConfigJSON), &config)
    
    // 3. 加载配置
    workflow.LoadWorkflowConfig(&config)
    
    // 4. 注册任务
    workflow.RegisterWorkflowTask("my_workflow", "task1",
        workflow.NewNormalTaskWorker(
            func(ctx context.Context, nodeContext *workflow.JSONContext) error {
                // 任务逻辑
                nodeContext.Set([]string{"result"}, "success")
                return nil
            },
            nil,
        ),
    )
    
    // 5. 创建实例
    instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
        WorkflowType: "my_workflow",
        BusinessID:   "TEST-001",
        Context:      map[string]any{"key": "value"},
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, instance)
}
```

### 示例 2: 测试 JSON 上下文

```go
func TestJSONExample(t *testing.T) {
    // 创建上下文
    ctx := workflow.NewJSONContextFromMap(map[string]any{
        "name": "张三",
        "age":  25,
    })
    
    // 读取值
    name, ok := ctx.GetString("name")
    assert.True(t, ok)
    assert.Equal(t, "张三", name)
    
    // 设置嵌套值
    ctx.Set([]string{"user", "email"}, "zhangsan@example.com")
    
    // 读取嵌套值
    email, ok := ctx.GetString("user", "email")
    assert.True(t, ok)
    assert.Equal(t, "zhangsan@example.com", email)
}
```

## 🔍 调试技巧

### 1. 查看详细日志

```bash
go test -v -run TestWorkflowExecution
```

### 2. 只运行失败的测试

```bash
go test -v -run TestFailed
```

### 3. 使用 dlv 调试器

```bash
# 安装 dlv
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试测试
dlv test -- -test.run TestWorkflowExecution
```

### 4. 添加断点

在测试代码中添加：

```go
import "runtime/debug"

func TestDebug(t *testing.T) {
    // 打印堆栈
    debug.PrintStack()
    
    // 你的测试代码
}
```

## ⚡ 性能优化

### 使用内存数据库

测试已经使用 SQLite 内存数据库 (`:memory:`)，速度很快。

### 并行测试

```go
func TestParallel(t *testing.T) {
    t.Parallel() // 标记为可并行
    
    // 测试代码
}
```

### 跳过慢速测试

```go
func TestSlow(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过慢速测试")
    }
    
    // 慢速测试代码
}
```

运行时使用 `-short` 标志：

```bash
go test -v -short ./...
```

## 📊 测试统计

当前测试覆盖：

- ✅ 工作流创建和执行
- ✅ JSON 上下文操作
- ✅ 异步任务处理
- ✅ 错误处理
- ✅ 并发安全性
- ✅ 上下文数据传递

## 🐛 常见问题

### Q: 测试失败 "no such table"

A: 确保在测试开始时调用了 `db.AutoMigrate()`

### Q: 并发测试不稳定

A: 使用适当的同步机制（如 `sync.WaitGroup`）

### Q: 测试运行很慢

A: 使用 `-parallel` 标志或 `-short` 跳过慢速测试

## 🔗 相关链接

- [主项目 README](../README.md)
- [Examples 示例](../examples/README.md)
- [Go Testing 文档](https://golang.org/pkg/testing/)
- [Testify 文档](https://github.com/stretchr/testify)

## 💡 贡献测试

欢迎添加更多测试用例！请遵循以下规范：

1. 使用清晰的测试名称
2. 每个测试只测试一个功能点
3. 使用 `t.Run()` 组织子测试
4. 添加必要的注释
5. 确保测试可以独立运行

示例：

```go
func TestMyFeature(t *testing.T) {
    t.Run("正常情况", func(t *testing.T) {
        // 测试正常情况
    })
    
    t.Run("边界情况", func(t *testing.T) {
        // 测试边界情况
    })
    
    t.Run("错误情况", func(t *testing.T) {
        // 测试错误情况
    })
}
```

---

**Happy Testing! 🎉**

