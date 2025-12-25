# Simple Workflow

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/blingmoon/simple-workflow)](https://goreportcard.com/report/github.com/blingmoon/simple-workflow)

一个轻量级、易用的 Go 工作流编排引擎，支持复杂的任务流程管理和持久化。

## ✨ 特性

- 🚀 **简单易用**：清晰的 API 设计，快速上手
- 🔄 **灵活编排**：支持顺序、分支、并行等多种工作流模式
- ⚡ **异步支持**：内置异步任务和状态检查机制
- 💾 **数据持久化**：支持 GORM，可使用 MySQL、PostgreSQL、SQLite 等数据库
- 🔒 **并发安全**：支持本地锁和分布式锁（Redis）
- 📊 **状态管理**：完善的工作流和任务状态跟踪
- 🔌 **易于集成**：零依赖核心，可与任何 Go 项目集成

## 📦 模块说明

### 公共 API（推荐使用）✅

```bash
# 稳定的公共 API，可安全导入
go get github.com/blingmoon/simple-workflow/workflow
```

**推荐导入**：
- `github.com/blingmoon/simple-workflow/workflow` - 核心工作流引擎

### 内部模块（受保护）🔒

项目使用 Go 的 `internal` 机制保护内部实现：

- ❌ **无法导入**: `github.com/blingmoon/simple-workflow/internal/tests` - 内部测试模块
- ❌ **无法导入**: `github.com/blingmoon/simple-workflow/internal/examples` - 内部示例模块

> **说明**：
> - Go 编译器会阻止外部项目导入 `internal/` 下的包（编译错误：`use of internal package not allowed`）
> - 示例代码位于 `internal/examples/`，可以复制到你的项目中使用
> - 测试代码位于 `internal/tests/`，仅用于项目内部质量保证

## 📦 安装

```bash
go get github.com/blingmoon/simple-workflow
```

## 🚀 快速开始

### 1. 初始化工作流服务

```go
package main

import (
    "github.com/blingmoon/simple-workflow/workflow"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    // 1. 初始化数据库
    db, err := gorm.Open(sqlite.Open("workflow.db"), &gorm.Config{})
    if err != nil {
        panic(err)
    }
    
    // 2. 自动迁移表结构
    db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
    
    // 3. 创建工作流服务
    workflowRepo := workflow.NewWorkflowRepo(db)
    workflowLock := workflow.NewLocalWorkflowLock()  // 本地锁，或使用 Redis 分布式锁
    workflowService := workflow.NewWorkflowService(workflowRepo, workflowLock)
}
```

### 2. 定义工作流配置

```go
import (
    "context"
    "encoding/json"
)

// 定义工作流结构：提交 -> 审核 -> 批准
workflowConfigJSON := `{
    "id": "approval_workflow",
    "name": "审批工作流",
    "nodes": [
        {
            "id": "submit",
            "name": "提交申请",
            "next_nodes": ["review"]
        },
        {
            "id": "review",
            "name": "审核",
            "next_nodes": ["approve"]
        },
        {
            "id": "approve",
            "name": "批准",
            "next_nodes": []
        }
    ]
}`

// 加载配置
workflowConfig := &workflow.WorkflowConfig{}
json.Unmarshal([]byte(workflowConfigJSON), workflowConfig)
workflow.LoadWorkflowConfig(workflowConfig)
```

### 3. 注册任务处理器

```go
// 注册"提交"任务
workflow.RegisterWorkflowTask("approval_workflow", "submit", 
    workflow.NewNormalTaskWorker(
        // Run 函数：同步执行
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 读取数据
            orderID, _ := nodeContext.GetString("workflow_context", "order_id")
            
            // 处理业务逻辑
            // ...
            
            // 写入数据
            nodeContext.Set([]string{"submit_time"}, time.Now().Unix())
            nodeContext.Set([]string{"status"}, "submitted")
            
            return nil
        },
        // AsynchronousWaitCheck 函数：异步检查（可选，传 nil 表示同步任务）
        nil,
    ),
)

// 注册"审核"任务（带异步检查）
workflow.RegisterWorkflowTask("approval_workflow", "review",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 启动审核流程
            nodeContext.Set([]string{"review_status"}, "pending")
            return nil
        },
        // 异步检查：等待外部审核结果
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            status, _ := nodeContext.GetString("review_status")
            if status != "approved" {
                // 返回错误表示还未完成，工作流会稍后重试
                return errors.New("waiting for approval")
            }
            return nil  // 返回 nil 表示异步任务完成
        },
    ),
)

// 注册"批准"任务
workflow.RegisterWorkflowTask("approval_workflow", "approve",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            nodeContext.Set([]string{"final_status"}, "approved")
            return nil
        },
        nil,
    ),
)
```

### 4. 创建和运行工作流实例

```go
// 创建工作流实例
workflowInstance, err := workflowService.CreateWorkflow(context.Background(), 
    &workflow.CreateWorkflowReq{
        WorkflowType: "approval_workflow",
        BusinessID:   "ORDER-001",
        Context: map[string]any{
            "order_id": "ORDER-001",
            "amount":   1000.00,
            "user_id":  "user123",
        },
    },
)

// 运行工作流
err = workflowService.RunWorkflow(context.Background(), workflowInstance.ID)
```

### 5. 处理异步任务

对于包含异步任务的工作流，需要定期调用 `RunWorkflow` 来检查异步任务状态：

```go
// 第一次运行：执行所有同步任务，启动异步任务
workflowService.RunWorkflow(ctx, workflowInstance.ID)

// 等待一段时间后，再次运行以检查异步任务
time.Sleep(5 * time.Second)
workflowService.RunWorkflow(ctx, workflowInstance.ID)

// 可以在定时任务或消息队列中定期调用
```

## 📖 核心概念

### 工作流配置

工作流由多个任务节点组成，每个节点可以指定下一个要执行的节点：

```go
type WorkflowConfig struct {
    ID    string                   `json:"id"`     // 工作流类型 ID
    Name  string                   `json:"name"`   // 工作流名称
    Nodes []*NodeDefinitionConfig  `json:"nodes"`  // 任务节点列表
}

type NodeDefinitionConfig struct {
    ID            string   `json:"id"`              // 节点 ID
    Name          string   `json:"name"`            // 节点名称
    NextNodes     []string `json:"next_nodes"`      // 下一个节点列表（空表示结束）
    FailMaxCount  int      `json:"fail_max_count"`  // 最大失败次数
    MaxWaitTimeTs int64    `json:"max_wait_time_ts"` // 最大等待时间（秒）
}
```

### 任务处理器

任务处理器包含两个函数：

1. **Run 函数**（必需）：同步执行任务逻辑
2. **AsynchronousWaitCheck 函数**（可选）：异步检查任务是否完成

```go
type WorkflowTaskNodeWorker interface {
    Run(ctx context.Context, nodeContext *JSONContext) error
    AsynchronousWaitCheck(ctx context.Context, nodeContext *JSONContext) error
}
```

### JSONContext 数据操作

`JSONContext` 提供了便捷的方法来读写任务上下文数据：

```go
// 写入数据
nodeContext.Set([]string{"user", "name"}, "Alice")
nodeContext.Set([]string{"status"}, "completed")

// 读取数据
name, ok := nodeContext.GetString("user", "name")
timestamp, ok := nodeContext.GetInt64("created_at")
success, ok := nodeContext.GetBool("is_success")

// 访问工作流全局上下文
orderID, ok := nodeContext.GetString("workflow_context", "order_id")
```

### NodeContext 数据流转机制

`NodeContext` 是节点间数据传递的核心机制。理解数据如何从前一个节点传递到下一个节点非常重要。

#### NodeContext 结构

每个节点的 `NodeContext` 包含以下主要部分：

```json
{
  "pre_node_context": {
    "前置节点1的TaskType": {
      "前置节点1输出的所有数据（已清理）"
    },
    "前置节点2的TaskType": {
      "前置节点2输出的所有数据（已清理）"
    }
  },
  "workflow_context": {
    "工作流全局上下文数据"
  },
  "当前节点自己写入的数据": "值"
}
```

#### 数据转换过程

**1. 节点初始化时的数据转换**

当工作流引擎创建新节点时，会自动进行以下转换：

```go
// 伪代码展示转换过程
func createNodeContext(preNodes []*WorkflowTaskNode, workflowContext *JSONContext) *JSONContext {
    preNodeAllContext := make(map[string]interface{})
    
    // 遍历所有前置节点
    for _, preNode := range preNodes {
        preNodeMap := preNode.NodeContext.ToMap()
        
        // 清理不需要传递的字段
        delete(preNodeMap, "pre_node_context")  // 不追溯到上层
        delete(preNodeMap, "workflow_context")  // 冗余字段
        delete(preNodeMap, "system")            // 系统参数不传递
        
        // 按前置节点的 TaskType 组织数据
        preNodeAllContext[preNode.TaskType] = preNodeMap
    }
    
    // 创建新节点的上下文
    return NewJSONContextFromMap(map[string]any{
        "pre_node_context": preNodeAllContext,
        "workflow_context": workflowContext.ToMap(),
    })
}
```

**2. 实际示例：数据流转**

假设有一个工作流：`submit` → `review` → `approve`

```go
// 节点1: submit（提交节点）
workflow.RegisterWorkflowTask("approval_workflow", "submit",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 写入提交节点的输出数据
            nodeContext.Set([]string{"submit_time"}, time.Now().Unix())
            nodeContext.Set([]string{"status"}, "submitted")
            nodeContext.Set([]string{"amount"}, 1000.0)
            
            // submit 节点的 NodeContext 结构：
            // {
            //   "workflow_context": {"order_id": "ORDER-001", ...},
            //   "submit_time": 1234567890,
            //   "status": "submitted",
            //   "amount": 1000.0
            // }
            return nil
        },
        nil,
    ),
)

// 节点2: review（审核节点）
workflow.RegisterWorkflowTask("approval_workflow", "review",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // ✅ 访问工作流全局上下文
            orderID, _ := nodeContext.GetString("workflow_context", "order_id")
            
            // ✅ 访问前置节点 submit 的输出数据
            submitTime, _ := nodeContext.GetInt64("pre_node_context", "submit", "submit_time")
            submitStatus, _ := nodeContext.GetString("pre_node_context", "submit", "status")
            amount, _ := nodeContext.GetFloat64("pre_node_context", "submit", "amount")
            
            fmt.Printf("审核订单 %s，提交时间: %d，状态: %s，金额: %.2f\n",
                orderID, submitTime, submitStatus, amount)
            
            // 写入审核节点的输出数据
            nodeContext.Set([]string{"review_time"}, time.Now().Unix())
            nodeContext.Set([]string{"reviewer"}, "manager")
            nodeContext.Set([]string{"review_result"}, "approved")
            
            // review 节点的 NodeContext 结构：
            // {
            //   "pre_node_context": {
            //     "submit": {
            //       "submit_time": 1234567890,
            //       "status": "submitted",
            //       "amount": 1000.0
            //     }
            //   },
            //   "workflow_context": {"order_id": "ORDER-001", ...},
            //   "review_time": 1234567900,
            //   "reviewer": "manager",
            //   "review_result": "approved"
            // }
            return nil
        },
        nil,
    ),
)

// 节点3: approve（批准节点）
workflow.RegisterWorkflowTask("approval_workflow", "approve",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // ✅ 访问工作流全局上下文
            orderID, _ := nodeContext.GetString("workflow_context", "order_id")
            
            // ✅ 访问前置节点 submit 的输出（跨节点访问）
            amount, _ := nodeContext.GetFloat64("pre_node_context", "submit", "amount")
            
            // ✅ 访问前置节点 review 的输出（直接前置节点）
            reviewer, _ := nodeContext.GetString("pre_node_context", "review", "reviewer")
            reviewResult, _ := nodeContext.GetString("pre_node_context", "review", "review_result")
            
            fmt.Printf("批准订单 %s，金额: %.2f，审核人: %s，审核结果: %s\n",
                orderID, amount, reviewer, reviewResult)
            
            // 写入批准节点的输出数据
            nodeContext.Set([]string{"approve_time"}, time.Now().Unix())
            nodeContext.Set([]string{"final_status"}, "approved")
            
            // approve 节点的 NodeContext 结构：
            // {
            //   "pre_node_context": {
            //     "submit": {
            //       "submit_time": 1234567890,
            //       "status": "submitted",
            //       "amount": 1000.0
            //     },
            //     "review": {
            //       "review_time": 1234567900,
            //       "reviewer": "manager",
            //       "review_result": "approved"
            //     }
            //   },
            //   "workflow_context": {"order_id": "ORDER-001", ...},
            //   "approve_time": 1234568000,
            //   "final_status": "approved"
            // }
            return nil
        },
        nil,
    ),
)
```

#### 数据访问模式总结

**访问工作流全局上下文**：
```go
// 格式：workflow_context.{字段名}
orderID, _ := nodeContext.GetString("workflow_context", "order_id")
amount, _ := nodeContext.GetFloat64("workflow_context", "amount")
```

**访问前置节点的输出**：
```go
// 格式：pre_node_context.{前置节点TaskType}.{字段名}
submitTime, _ := nodeContext.GetInt64("pre_node_context", "submit", "submit_time")
reviewer, _ := nodeContext.GetString("pre_node_context", "review", "reviewer")
```

**访问当前节点自己写入的数据**：
```go
// 直接访问，不需要前缀
currentStatus, _ := nodeContext.GetString("status")
```

#### 重要注意事项

1. **数据清理规则**：前置节点的 `pre_node_context`、`workflow_context`、`system` 字段会被自动删除，不会传递到下一层
2. **数据组织方式**：前置节点的数据按 `TaskType` 组织在 `pre_node_context` 中
3. **跨节点访问**：后续节点可以访问所有前置节点的输出，不仅仅是直接前置节点
4. **数据隔离**：每个节点的输出数据是独立的，不会相互覆盖
5. **系统字段**：`system` 字段包含系统错误信息等，不会传递给下游节点

### 特殊错误类型

工作流引擎提供了一些特殊的错误类型，用于精确控制工作流的执行行为：

#### 1. ErrorWorkflowTaskInstanceNotReady（任务未准备好）

**用途**：表示任务当前阶段还没有准备好，需要稍后重试。

**场景**：
- 等待外部审核结果
- 等待第三方 API 响应
- 等待定时任务触发

**示例**：
```go
workflow.RegisterWorkflowTask("approval_workflow", "review",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 发起审核请求
            nodeContext.Set([]string{"review_status"}, "pending")
            return nil
        },
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 异步检查审核结果
            status, _ := nodeContext.GetString("review_status")
            if status == "pending" {
                // 返回此错误，工作流会保持任务为 pending 状态，稍后重试
                return workflow.ErrorWorkflowTaskInstanceNotReady
            }
            return nil  // 审核完成，继续执行
        },
    ),
)
```

**行为**：
- 任务保持 `pending` 状态
- 工作流继续运行，但不会推进此任务
- 下次运行工作流时会重新检查

#### 2. ErrorWorkflowTaskFailedWithContinue（失败但继续）

**用途**：任务失败，但不影响工作流继续执行，可以当作另一种形式的"完成"。

**场景**：
- 非关键的通知任务失败（如发送邮件）
- 可选的数据采集任务
- 降级场景处理

**示例**：
```go
workflow.RegisterWorkflowTask("order_workflow", "send_notification",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 尝试发送通知
            err := sendEmail(...)
            if err != nil {
                // 发送失败，但不影响订单流程
                nodeContext.Set([]string{"notification_sent"}, false)
                return workflow.ErrorWorkflowTaskFailedWithContinue
            }
            nodeContext.Set([]string{"notification_sent"}, true)
            return nil
        },
        nil,
    ),
)
```

**行为**：
- 任务标记为完成（虽然失败了）
- 工作流继续执行后续节点
- 可以通过上下文查看任务实际执行结果

#### 3. ErrWorkflowTaskFailedWithFailed（失败并终止）

**用途**：任务失败，整个工作流应该终止，状态变为 `failed`。

**场景**：
- 关键参数缺失或无效
- 不可恢复的业务错误
- 数据一致性检查失败

**示例**：
```go
workflow.RegisterWorkflowTask("payment_workflow", "validate_account",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            accountID, ok := nodeContext.GetString("workflow_context", "account_id")
            if !ok || accountID == "" {
                // 账户ID缺失，无法继续，终止工作流
                return workflow.ErrWorkflowTaskFailedWithFailed
            }
            
            // 验证账户
            if !isValidAccount(accountID) {
                // 账户无效，终止工作流
                return workflow.ErrWorkflowTaskFailedWithFailed
            }
            
            return nil
        },
        nil,
    ),
)
```

**行为**：
- 任务标记为 `failed`
- 工作流状态变为 `failed`
- 停止执行后续节点
- 需要人工介入或重启工作流

#### 4. ErrWorkBussinessCriticalError（业务严重错误）

**用途**：用于标识需要人工介入的严重业务错误，通常用于报警和监控。

**场景**：
- 数据不一致
- 重要业务流程异常
- 需要立即处理的错误

**示例**：
```go
import "github.com/pkg/errors"

workflow.RegisterWorkflowTask("reconciliation_workflow", "check_balance",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            expected := getExpectedBalance()
            actual := getActualBalance()
            
            if expected != actual {
                // 余额不一致，严重错误，需要报警
                return errors.Wrapf(
                    workflow.ErrWorkBussinessCriticalError,
                    "balance mismatch: expected=%f, actual=%f",
                    expected, actual,
                )
            }
            return nil
        },
        nil,
    ),
)
```

**行为**：
- 任务失败
- 日志记录为 ERROR 级别
- 触发报警系统（需要在调度系统中配置）
- 可以通过 `errors.Is()` 识别此类错误

#### 5. ErrWorkBussinessWarningError（业务警告错误）

**用途**：用于标识需要关注但不严重的业务错误，记录为警告级别。

**场景**：
- 降级服务使用
- 性能指标异常
- 非关键功能异常

**示例**：
```go
workflow.RegisterWorkflowTask("order_workflow", "calculate_discount",
    workflow.NewNormalTaskWorker(
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            discount, err := getDiscountFromService()
            if err != nil {
                // 折扣服务失败，使用默认值，记录警告
                discount = 0.0
                nodeContext.Set([]string{"discount"}, discount)
                return errors.Wrapf(
                    workflow.ErrWorkBussinessWarningError,
                    "discount service failed, using default: %v",
                    err,
                )
            }
            nodeContext.Set([]string{"discount"}, discount)
            return nil
        },
        nil,
    ),
)
```

**行为**：
- 任务失败（或根据配置决定）
- 日志记录为 WARN 级别
- 可用于监控趋势
- 不会触发紧急报警

### 错误处理最佳实践

```go
// 1. 检查特定错误类型
if errors.Is(err, workflow.ErrorWorkflowTaskInstanceNotReady) {
    // 任务未准备好，继续等待
}

// 2. 包装业务错误
if criticalError {
    return errors.Wrapf(
        workflow.ErrWorkBussinessCriticalError,
        "详细错误信息: %v", err,
    )
}

// 3. 根据场景选择合适的错误
func handleTask(ctx context.Context, nodeContext *workflow.JSONContext) error {
    if missingRequiredData {
        // 缺少必需数据，终止工作流
        return workflow.ErrWorkflowTaskFailedWithFailed
    }
    
    if externalServiceNotReady {
        // 外部服务未准备好，稍后重试
        return workflow.ErrorWorkflowTaskInstanceNotReady
    }
    
    if optionalFeatureFailed {
        // 可选功能失败，继续执行
        return workflow.ErrorWorkflowTaskFailedWithContinue
    }
    
    return nil  // 成功
}
```

## 🗄️ 数据持久化

### 使用不同的数据库

```go
// SQLite
import "gorm.io/driver/sqlite"
db, _ := gorm.Open(sqlite.Open("workflow.db"), &gorm.Config{})

// MySQL
import "gorm.io/driver/mysql"
dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4"
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

// PostgreSQL
import "gorm.io/driver/postgres"
dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920"
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

### 数据表结构

工作流引擎会自动创建两张表：

- `workflow_instance`：工作流实例表
- `task_instance`：任务实例表

## 🔒 并发控制

### 本地锁（单机）

```go
workflowLock := workflow.NewLocalWorkflowLock()
```

### Redis 分布式锁（多机）

```go
import "github.com/redis/go-redis/v9"

redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

workflowLock := workflow.NewRedisWorkflowLock(redisClient)
```

## 📊 工作流状态

### 工作流实例状态

- `init` - 初始化
- `running` - 运行中
- `completed` - 已完成
- `failed` - 失败
- `cancelled` - 已取消

### 任务节点状态

- `running` - 运行中
- `pending` - 等待中（异步任务）
- `finishing` - 完成中
- `completed` - 已完成
- `failed` - 失败
- `cancelled` - 已取消

## 🎯 高级功能

### 重启失败的任务

```go
// 重启指定任务节点
err := workflowService.RestartWorkflowNode(ctx, &workflow.RestartWorkflowNodeParams{
    WorkflowInstanceID: instanceID,
    TaskType:          "review",
    IsAsynchronous:    false,  // 是否异步重启
})

// 重启整个工作流实例
err := workflowService.RestartWorkflowInstance(ctx, &workflow.RestartWorkflowParams{
    WorkflowInstanceID: instanceID,
    Context:           newContext,  // 可选：更新上下文
    IsRun:             true,        // 是否立即运行
})
```

### 添加外部事件

```go
// 为任务节点添加外部事件（如审核结果）
err := workflowService.AddNodeExternalEvent(ctx, &workflow.AddNodeExternalEventParams{
    WorkflowInstanceID: instanceID,
    TaskType:          "review",
    NodeEvent: &workflow.NodeEvent{
        EventTs:      time.Now().Unix(),
        EventContent: map[string]any{"result": "approved"},
    },
})
```

### 查询工作流状态

```go
// 查询工作流实例详情
details, err := workflowService.QueryWorkflowInstanceDetail(ctx, 
    &workflow.QueryWorkflowInstanceParams{
        WorkflowInstanceID: &instanceID,
    },
)

// 统计工作流实例数量
count, err := workflowService.CountWorkflowInstance(ctx, 
    &workflow.QueryWorkflowInstanceParams{
        WorkflowType: "approval_workflow",
        Status:       "running",
    },
)
```

## 📚 完整示例

查看 [examples/with-sqlite](examples/with-sqlite) 目录获取完整的可运行示例：

```bash
# 运行完整示例
cd examples/with-sqlite
go run main.go

# 运行测试
go test -v
```

示例包含：
- ✅ 基础工作流创建和执行
- ✅ 异步任务处理
- ✅ 多分支工作流
- ✅ 批量创建工作流实例
- ✅ 数据持久化和查询

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 查看测试覆盖率
go test -cover ./...

# 运行特定包的测试
go test ./workflow -v
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🔗 相关链接

- [完整示例](examples/)
- [API 文档](https://pkg.go.dev/github.com/blingmoon/simple-workflow)
- [问题反馈](https://github.com/blingmoon/simple-workflow/issues)
