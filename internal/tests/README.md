# Simple Workflow 单元测试

这是一个独立的测试模块，用于测试 simple-workflow 包的各项功能。

## 📋 目录结构

```
tests/
├── go.mod                       # 独立的 go.mod
├── README.md                    # 本文件
├── workflow_basic_test.go       # 基础工作流测试
├── json_context_simple_test.go  # JSON 上下文测试
└── integration_test.go          # 集成测试
```

## 🚀 运行测试

### 运行所有测试

```bash
cd tests
go test -v ./...
```

### 运行特定测试

```bash
# 基础工作流测试
go test -v -run TestWorkflowCreationBasic
go test -v -run TestWorkflowExecution
go test -v -run TestWorkflowQuery
go test -v -run TestWorkflowContext

# JSON 上下文测试
go test -v -run TestJSONContextSimple

# 集成测试
go test -v -run TestCompleteWorkflowScenario
go test -v -run TestAsyncWorkflow
go test -v -run TestErrorHandling
go test -v -run TestConcurrentExecution
```

### 运行性能测试

```bash
go test -v -bench=. -benchmem
```

### 查看测试覆盖率

```bash
go test -v -cover ./...
```

### 生成覆盖率报告

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 📦 测试模块说明

### 1. workflow_basic_test.go - 基础工作流测试

测试工作流的基本功能：
- ✅ 工作流服务创建
- ✅ 工作流实例创建
- ✅ 工作流执行（单任务和多任务）
- ✅ 工作流查询和统计
- ✅ 上下文数据传递
- ✅ 性能基准测试

### 2. json_context_simple_test.go - JSON 上下文测试

测试 JSON 上下文的操作：
- ✅ 从 JSON 字符串创建
- ✅ 从 map 创建
- ✅ 数据存取（字符串、整数、浮点数、布尔值）
- ✅ 嵌套路径操作
- ✅ 序列化和反序列化
- ✅ 克隆操作

### 3. integration_test.go - 集成测试

端到端的集成测试：
- ✅ 完整工作流场景（订单处理）
- ✅ 异步工作流（带异步检查）
- ✅ 错误处理（任务失败、特殊错误）
- ✅ 并发执行测试

## 🔧 依赖管理

本测试模块使用独立的 `go.mod`，通过 `replace` 指令引用主项目：

```go
replace github.com/blingmoon/simple-workflow => ../
```

这样做的好处：
1. **隔离依赖** - 测试依赖不会污染主项目
2. **独立版本** - 可以使用不同版本的测试库
3. **清晰结构** - 测试代码与业务代码分离
4. **灵活开发** - 可以独立更新测试

## 🎯 测试最佳实践

1. **使用 testify** - 使用 assert 和 require 简化断言
2. **清理资源** - 使用 `defer` 或 `t.Cleanup()` 清理测试资源
3. **表格驱动** - 使用表格驱动测试提高覆盖率
4. **子测试** - 使用 `t.Run()` 组织相关测试
5. **并发测试** - 使用 `t.Parallel()` 加速测试

## 📝 添加新测试

创建新测试文件：

```go
package tests

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/blingmoon/simple-workflow/workflow"
)

func TestYourFeature(t *testing.T) {
	t.Run("test case 1", func(t *testing.T) {
		// 测试逻辑
		assert.NotNil(t, result)
	})
}
```

## 🐛 调试测试

### 启用详细日志

```bash
go test -v -args -test.v
```

### 调试单个测试

```bash
go test -v -run TestSpecificTest
```

### 使用 dlv 调试器

```bash
dlv test -- -test.run TestSpecificTest
```

## 📊 CI/CD 集成

在 CI 中运行测试：

```yaml
- name: Run Tests
  run: |
    cd tests
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

## 🔗 相关链接

- [主项目 README](../README.md)
- [Examples 示例](../examples/README.md)
- [Go Testing 文档](https://golang.org/pkg/testing/)
- [Testify 文档](https://github.com/stretchr/testify)

