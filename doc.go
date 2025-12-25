// Package workflow 提供工作流编排功能。
//
// 这是一个轻量级、易用的 Go 工作流编排引擎，支持复杂的任务流程管理和持久化。
//
// 主要特性：
//   - 简单易用：清晰的 API 设计，快速上手
//   - 灵活编排：支持顺序、分支、并行等多种工作流模式
//   - 异步支持：内置异步任务和状态检查机制
//   - 数据持久化：支持 GORM，可使用 MySQL、PostgreSQL、SQLite 等数据库
//   - 并发安全：支持本地锁和分布式锁（Redis）
//   - 状态管理：完善的工作流和任务状态跟踪
//
// 基础使用示例:
//
//	package main
//
//	import (
//	    "context"
//	    "encoding/json"
//	    "time"
//
//	    "github.com/blingmoon/simple-workflow/workflow"
//	    "gorm.io/driver/sqlite"
//	    "gorm.io/gorm"
//	)
//
//	func main() {
//	    // 1. 初始化数据库
//	    db, _ := gorm.Open(sqlite.Open("workflow.db"), &gorm.Config{})
//	    db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
//
//	    // 2. 创建工作流服务
//	    workflowRepo := workflow.NewWorkflowRepo(db)
//	    workflowLock := workflow.NewLocalWorkflowLock()
//	    workflowService := workflow.NewWorkflowService(workflowRepo, workflowLock)
//
//	    // 3. 定义工作流配置
//	    workflowConfigJSON := `{
//	        "id": "approval_workflow",
//	        "name": "审批工作流",
//	        "nodes": [
//	            {"id": "submit", "name": "提交申请", "next_nodes": ["review"]},
//	            {"id": "review", "name": "审核", "next_nodes": ["approve"]},
//	            {"id": "approve", "name": "批准", "next_nodes": []}
//	        ]
//	    }`
//	    workflowConfig := &workflow.WorkflowConfig{}
//	    json.Unmarshal([]byte(workflowConfigJSON), workflowConfig)
//	    workflow.LoadWorkflowConfig(workflowConfig)
//
//	    // 4. 注册任务处理器
//	    workflow.RegisterWorkflowTask("approval_workflow", "submit",
//	        workflow.NewNormalTaskWorker(
//	            func(ctx context.Context, nodeContext *workflow.JSONContext) error {
//	                nodeContext.Set([]string{"submit_time"}, time.Now().Unix())
//	                return nil
//	            },
//	            nil,
//	        ),
//	    )
//
//	    // 5. 创建和运行工作流实例
//	    instance, _ := workflowService.CreateWorkflow(context.Background(),
//	        &workflow.CreateWorkflowReq{
//	            WorkflowType: "approval_workflow",
//	            BusinessID:   "ORDER-001",
//	            Context:      map[string]any{"order_id": "ORDER-001"},
//	        },
//	    )
//	    workflowService.RunWorkflow(context.Background(), instance.ID)
//	}
//
// NodeContext 数据流转机制：
//
// NodeContext 是节点间数据传递的核心机制。每个节点的 NodeContext 包含：
//
//   - pre_node_context: 所有前置节点的输出数据（按 TaskType 组织）
//   - workflow_context: 工作流全局上下文数据
//   - 当前节点自己写入的数据
//
// 数据访问示例：
//
//	// 访问工作流全局上下文
//	orderID, _ := nodeContext.GetString("workflow_context", "order_id")
//
//	// 访问前置节点的输出（格式：pre_node_context.{前置节点TaskType}.{字段名}）
//	submitTime, _ := nodeContext.GetInt64("pre_node_context", "submit", "submit_time")
//
//	// 写入当前节点的数据
//	nodeContext.Set([]string{"review_time"}, time.Now().Unix())
//
// 数据转换规则：
//
// 当工作流引擎创建新节点时，会自动：
//   - 收集所有前置节点的 NodeContext
//   - 清理前置节点的 pre_node_context、workflow_context、system 字段
//   - 按前置节点的 TaskType 组织数据到 pre_node_context 中
//   - 创建包含 pre_node_context 和 workflow_context 的新 NodeContext
//
// 更多示例和文档请访问: https://github.com/blingmoon/simple-workflow
package workflow
