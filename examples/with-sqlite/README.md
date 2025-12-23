# Workflow + SQLite 完整示例

这是一个完整的 simple-workflow 使用示例，展示如何使用 SQLite 作为存储后端。

## 功能特性

- ✅ 使用 SQLite 作为数据持久化存储
- ✅ 使用 GORM 作为 ORM 框架
- ✅ 完整的工作流定义、注册、创建和执行流程
- ✅ 包含同步和异步任务节点
- ✅ 演示 JSONContext 的使用
- ✅ 完整的单元测试

## 文件结构

```
with-sqlite/
├── main.go           # 可执行示例程序
├── workflow_test.go  # 单元测试
└── README.md         # 本文档
```

## 快速开始

### 运行示例程序

```bash
cd examples/with-sqlite
go run main.go
```

**输出示例**：
```
=== Simple Workflow + SQLite 完整示例 ===

正在初始化数据库表...
✓ Workflow 服务创建成功

正在加载工作流配置...
✓ 工作流配置加载成功

正在注册任务节点...
✓ 所有任务节点注册成功

正在创建工作流实例...
✓ 工作流实例创建成功 (ID: 1)

正在运行工作流...
  [提交] 执行中...
  [提交] 完成 ✓
  [审核] 执行中...
  [审核] 完成 ✓

等待异步任务完成...
正在完成剩余任务...
  [审核] 检查审核状态 (提交时间: 2024-12-23T14:30:00Z)
  [批准] 执行中...
  [批准] 完成 ✓

✅ 工作流执行完成！
```

程序运行后会在当前目录生成 `workflow.db` 数据库文件。

### 运行测试

```bash
go test -v
```

**测试内容**：
- `TestWorkflowWithSQLite` - 测试完整的工作流执行流程
- `TestMultipleWorkflows` - 测试创建和运行多个工作流实例

## 代码解析

### 1. 初始化数据库

```go
// 连接 SQLite 数据库
db, err := gorm.Open(sqlite.Open("workflow.db"), &gorm.Config{})

// 自动迁移表结构
db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
```

### 2. 创建 Workflow 服务

```go
workflowRepo := workflow.NewWorkflowRepo(db)
workflowLock := workflow.NewLocalWorkflowLock()
workflowService := workflow.NewWorkflowService(workflowRepo, workflowLock)
```

### 3. 定义工作流配置

```go
workflowConfig := &workflow.WorkflowConfig{
    ID:   "approval_workflow",
    Name: "审批工作流",
    Nodes: []*workflow.NodeDefinitionConfig{
        {
            ID:        "submit",
            Name:      "提交申请",
            NextNodes: []string{"review"},
        },
        {
            ID:        "review",
            Name:      "审核",
            NextNodes: []string{"approve"},
        },
        {
            ID:        "approve",
            Name:      "批准",
            NextNodes: []string{},
        },
    },
}

workflow.LoadWorkflowConfig(workflowConfig)
```

### 4. 注册任务处理器

```go
// 使用 JSONContext 进行数据读写
workflow.RegisterWorkflowTask("approval_workflow", "submit", 
    workflow.NewNormalTaskWorker(
        // Run 函数 - 同步执行
        func(ctx context.Context, nodeContext *workflow.JSONContext) error {
            // 写入数据
            nodeContext.Set([]string{"submit_time"}, time.Now().Format(time.RFC3339))
            nodeContext.Set([]string{"status"}, "submitted")
            return nil
        },
        // AsynchronousWaitCheck 函数 - 异步检查（可选）
        nil,
    ),
)
```

### 5. 创建和运行工作流

```go
// 创建工作流实例
workflowInstance, err := workflowService.CreateWorkflow(context.Background(), 
    &workflow.CreateWorkflowReq{
        WorkflowType: "approval_workflow",
        BusinessID:   "ORDER-2024-001",
        Context: map[string]any{
            "order_id": "ORDER-2024-001",
            "amount":   1000.00,
        },
    },
)

// 运行工作流
err = workflowService.RunWorkflow(context.Background(), workflowInstance.ID)
```

## JSONContext 使用示例

### 写入数据

```go
func(ctx context.Context, nodeContext *workflow.JSONContext) error {
    // 设置顶层字段
    nodeContext.Set([]string{"status"}, "processing")
    
    // 设置嵌套字段（自动创建中间路径）
    nodeContext.Set([]string{"result", "success"}, true)
    nodeContext.Set([]string{"result", "message"}, "操作成功")
    
    return nil
}
```

### 读取数据

```go
func(ctx context.Context, nodeContext *workflow.JSONContext) error {
    // 读取字符串
    status, ok := nodeContext.GetString("status")
    if !ok {
        return errors.New("status not found")
    }
    
    // 读取整数
    timestamp, ok := nodeContext.GetInt64("workflow_context", "created_time")
    if !ok {
        return errors.New("created_time not found")
    }
    
    // 读取布尔值
    success, ok := nodeContext.GetBool("result", "success")
    
    return nil
}
```

## 查看数据库

运行示例后，可以使用 SQLite 命令行工具查看数据：

```bash
# 打开数据库
sqlite3 workflow.db

# 查看工作流实例
SELECT * FROM workflow_instance;

# 查看任务实例
SELECT * FROM task_instance;

# 查看特定工作流的所有任务
SELECT 
    ti.id,
    ti.task_type,
    ti.status,
    ti.created_at,
    ti.updated_at
FROM task_instance ti
WHERE ti.workflow_instance_id = 1
ORDER BY ti.id;
```

## 工作流状态

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
- `restarting` - 重启中

## 高级功能

### 异步任务

某些任务需要等待外部事件或条件，可以使用异步检查：

```go
workflow.NewNormalTaskWorker(
    // Run - 启动任务
    func(ctx context.Context, nodeContext *workflow.JSONContext) error {
        // 发起异步操作（如发送 HTTP 请求）
        nodeContext.Set([]string{"async_started"}, true)
        return nil
    },
    // AsynchronousWaitCheck - 检查任务是否完成
    func(ctx context.Context, nodeContext *workflow.JSONContext) error {
        // 检查异步操作是否完成
        // 如果未完成，返回 ErrorWorkflowTaskInstanceNotReady
        // 如果完成，返回 nil
        
        startTime, _ := nodeContext.GetInt64("async_started_at")
        if time.Now().Unix() < startTime + 60 {
            return workflow.ErrorWorkflowTaskInstanceNotReady
        }
        return nil
    },
)
```

### 任务重试

可以为任务设置最大失败次数：

```json
{
    "id": "risky_task",
    "name": "可能失败的任务",
    "next_nodes": ["next_task"],
    "fail_max_count": 3
}
```

### 任务超时

可以为任务设置最大等待时间：

```json
{
    "id": "time_sensitive_task",
    "name": "时间敏感任务",
    "next_nodes": ["next_task"],
    "max_wait_time_ts": 3600
}
```

## 依赖说明

这个示例使用了以下依赖（只在 examples 中，不影响主项目）：

```go
require (
    github.com/stretchr/testify v1.11.1  // 测试框架
    gorm.io/driver/sqlite v1.6.0         // SQLite 驱动
)
```

这些依赖**不会**添加到主项目的 `go.mod` 中，详见 [依赖独立性说明](../VERIFY.md)。

## 常见问题

### Q: 为什么需要运行两次 RunWorkflow？

A: 第一次运行启动同步任务和异步任务的 Run 阶段。异步任务需要等待一段时间，所以第二次运行时会执行 AsynchronousWaitCheck 来检查异步任务是否完成。

### Q: 如何查看工作流执行日志？

A: 可以启用日志输出，或者查询数据库中的 `task_instance` 表查看每个任务的状态。

### Q: 可以使用其他数据库吗？

A: 可以！只需要更换 GORM 驱动即可，例如：
- MySQL: `gorm.io/driver/mysql`
- PostgreSQL: `gorm.io/driver/postgres`
- SQL Server: `gorm.io/driver/sqlserver`

### Q: 如何清理数据库？

A: 删除 `workflow.db` 文件即可：
```bash
rm workflow.db
```

## 下一步

- 查看 [主项目文档](../../README.md) 了解更多功能
- 查看 [API 文档](../../doc.go) 了解接口详情
- 尝试修改工作流配置创建自己的工作流

---

**提示**：这是一个演示示例，生产环境使用时需要：
1. 添加适当的错误处理
2. 配置日志系统
3. 实现分布式锁（如使用 Redis）
4. 添加监控和告警

