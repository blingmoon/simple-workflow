package commonregister

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
)

func RegisterApprovalInstanceTask(service workflow.WorkflowService) error {
	// 1. 定义工作流配置
	// 工作流结构：提交 -> 审核 -> 批准
	workflowConfigJson := `{
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

	workflowConfig := &workflow.WorkflowConfig{}
	if err := json.Unmarshal([]byte(workflowConfigJson), workflowConfig); err != nil {
		return errors.Wrap(err, "unmarshal workflow config failed")
	}

	// 2. 加载工作流配置
	if err := workflow.LoadWorkflowConfig(workflowConfig); err != nil {
		return errors.Wrap(err, "load workflow config failed")
	}

	// 3. 注册任务节点处理器
	// 提交申请节点
	err := workflow.RegisterWorkflowTask("approval_workflow", "submit", workflow.NewNormalTaskWorker(
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			fmt.Println("  [提交] 执行中...")
			nodeContext.Set([]string{"submit_time"}, time.Now().Format(time.RFC3339))
			nodeContext.Set([]string{"status"}, "submitted")
			fmt.Println("  [提交] 完成 ✓")
			return nil
		},
		nil,
	))
	if err != nil {
		return errors.Wrap(err, "register submit task failed")
	}

	// 审核节点（包含异步检查）
	err = workflow.RegisterWorkflowTask("approval_workflow", "review", workflow.NewNormalTaskWorker(
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			fmt.Println("  [审核] 执行中...")
			submitTime, ok := nodeContext.GetString("pre_node_context", "submit", "submit_time")
			if !ok {
				return errors.New("submit_time not found")
			}

			nodeContext.Set([]string{"submit_time"}, submitTime)
			nodeContext.Set([]string{"review_time"}, time.Now().Format(time.RFC3339))
			nodeContext.Set([]string{"reviewer"}, "manager")

			fmt.Println("  [审核] 完成 ✓")
			return nil
		},
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			// 异步检查：模拟等待审核完成
			submitTime, ok := nodeContext.GetString("pre_node_context", "submit", "submit_time")
			if !ok {
				return errors.New("submit_time not found")
			}
			fmt.Printf("  [审核-异步检查] 验证提交时间: %s\n", submitTime)
			return nil
		},
	))
	if err != nil {
		return errors.Wrap(err, "register review task failed")
	}

	// 批准节点
	err = workflow.RegisterWorkflowTask("approval_workflow", "approve", workflow.NewNormalTaskWorker(
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			fmt.Println("  [批准] 执行中...")
			nodeContext.Set([]string{"approve_time"}, time.Now().Format(time.RFC3339))
			nodeContext.Set([]string{"final_status"}, "approved")
			fmt.Println("  [批准] 完成 ✓")
			return nil
		},
		nil,
	))
	if err != nil {
		return errors.Wrap(err, "register approve task failed")
	}
	return nil
}
