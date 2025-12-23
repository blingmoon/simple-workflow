package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== Simple Workflow + SQLite 完整示例 ===")
	fmt.Println()

	// 1. 初始化 SQLite 数据库
	db, err := gorm.Open(sqlite.Open("workflow.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// 2. 自动迁移数据库表
	fmt.Println("正在初始化数据库表...")
	if err := db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{}); err != nil {
		panic(err)
	}

	// 3. 创建 workflow 服务
	workflowRepo := workflow.NewWorkflowRepo(db)
	workflowLock := workflow.NewLocalWorkflowLock()
	workflowService := workflow.NewWorkflowService(workflowRepo, workflowLock)
	fmt.Println("✓ Workflow 服务创建成功")
	fmt.Println()

	// 运行示例1：审批工作流
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("示例 1: 审批工作流（包含异步检查）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runApprovalWorkflow(workflowService)
	fmt.Println()

	// 运行示例2：复杂工作流
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("示例 2: 复杂工作流（多分支 + 异步任务）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runComplexWorkflow(workflowService)
	fmt.Println()

	// 运行示例3：多个工作流实例
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("示例 3: 创建多个工作流实例")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	runMultipleWorkflows(workflowService)
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 所有示例执行完成！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("📁 数据库文件: workflow.db")
	fmt.Println()
	fmt.Println("💡 你可以使用 SQLite 客户端查看数据：")
	fmt.Println("   $ sqlite3 workflow.db")
	fmt.Println("   sqlite> SELECT * FROM workflow_instance;")
	fmt.Println("   sqlite> SELECT * FROM task_instance;")
}

// runApprovalWorkflow 运行审批工作流示例
func runApprovalWorkflow(workflowService workflow.WorkflowService) {

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
		panic(err)
	}

	// 2. 加载工作流配置
	if err := workflow.LoadWorkflowConfig(workflowConfig); err != nil {
		panic(err)
	}
	fmt.Println("✓ 工作流配置加载成功")

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
		panic(err)
	}

	// 审核节点（包含异步检查）
	err = workflow.RegisterWorkflowTask("approval_workflow", "review", workflow.NewNormalTaskWorker(
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			fmt.Println("  [审核] 执行中...")
			nodeContext.Set([]string{"review_time"}, time.Now().Format(time.RFC3339))
			nodeContext.Set([]string{"reviewer"}, "manager")
			fmt.Println("  [审核] 完成 ✓")
			return nil
		},
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			// 异步检查：模拟等待审核完成
			submitTime, ok := nodeContext.GetString("submit_time")
			if !ok {
				return errors.New("submit_time not found")
			}
			fmt.Printf("  [审核-异步检查] 验证提交时间: %s\n", submitTime)
			return nil
		},
	))
	if err != nil {
		panic(err)
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
		panic(err)
	}

	fmt.Println("✓ 任务节点注册成功")

	// 4. 创建工作流实例
	workflowInstance, err := workflowService.CreateWorkflow(context.Background(), &workflow.CreateWorkflowReq{
		WorkflowType: "approval_workflow",
		BusinessID:   "ORDER-2024-001",
		Context: map[string]any{
			"order_id":     "ORDER-2024-001",
			"amount":       1000.00,
			"created_time": time.Now().Unix(),
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("✓ 工作流实例创建成功 (ID: %d)\n", workflowInstance.ID)

	// 5. 第一次运行工作流
	fmt.Println("正在运行工作流...")
	if err := workflowService.RunWorkflow(context.Background(), workflowInstance.ID); err != nil {
		panic(err)
	}

	// 6. 等待异步任务
	fmt.Println("等待异步任务完成（2秒）...")
	time.Sleep(2 * time.Second)

	// 7. 第二次运行以完成异步检查
	fmt.Println("继续运行工作流（完成异步检查）...")
	if err := workflowService.RunWorkflow(context.Background(), workflowInstance.ID); err != nil {
		panic(err)
	}

	fmt.Printf("✅ 审批工作流执行完成！(实例 ID: %d)\n", workflowInstance.ID)
}

// runComplexWorkflow 运行复杂工作流示例（多分支 + 异步任务）
func runComplexWorkflow(workflowService workflow.WorkflowService) {
	// 1. 定义复杂工作流配置
	// 流程：A -> B -> C
	//      A1 -> B, B1
	workflowConfigJson := `{
		"id": "complex_workflow",
		"name": "复杂工作流",
		"nodes": [
			{
				"id": "A",
				"name": "任务A",
				"next_nodes": ["B"]
			},
			{
				"id": "A1",
				"name": "任务A1（多分支）",
				"next_nodes": ["B", "B1"]
			},
			{
				"id": "B",
				"name": "任务B",
				"next_nodes": ["C"]
			},
			{
				"id": "B1",
				"name": "任务B1（异步）",
				"next_nodes": []
			},
			{
				"id": "C",
				"name": "任务C",
				"next_nodes": []
			}
		]
	}`

	workflowConfig := &workflow.WorkflowConfig{}
	if err := json.Unmarshal([]byte(workflowConfigJson), workflowConfig); err != nil {
		panic(err)
	}

	// 2. 加载配置
	if err := workflow.LoadWorkflowConfig(workflowConfig); err != nil {
		panic(err)
	}
	fmt.Println("✓ 复杂工作流配置加载成功")

	// 3. 注册任务节点
	tasks := []struct {
		id   string
		name string
	}{
		{"A", "任务A"},
		{"A1", "任务A1"},
		{"B", "任务B"},
		{"C", "任务C"},
	}

	for _, task := range tasks {
		taskID := task.id
		taskName := task.name
		err := workflow.RegisterWorkflowTask("complex_workflow", taskID, workflow.NewNormalTaskWorker(
			func(ctx context.Context, nodeContext *workflow.JSONContext) error {
				fmt.Printf("  [%s] 执行中...\n", taskName)
				nodeContext.Set([]string{"name"}, taskName)
				nodeContext.Set([]string{"completed"}, true)
				nodeContext.Set([]string{"exec_time"}, time.Now().Format(time.RFC3339))
				fmt.Printf("  [%s] 完成 ✓\n", taskName)
				return nil
			},
			nil,
		))
		if err != nil {
			panic(err)
		}
	}

	// B1 任务（包含异步检查）
	err := workflow.RegisterWorkflowTask("complex_workflow", "B1", workflow.NewNormalTaskWorker(
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			fmt.Println("  [任务B1-异步] 启动异步任务...")
			nodeContext.Set([]string{"name"}, "任务B1")
			nodeContext.Set([]string{"async_started"}, true)
			fmt.Println("  [任务B1-异步] 异步任务已启动 ✓")
			return nil
		},
		func(ctx context.Context, nodeContext *workflow.JSONContext) error {
			// 异步检查：验证冷却时间
			curTime := time.Now().Unix()
			createdTime, ok := nodeContext.GetInt64("workflow_context", "created_time")
			if !ok {
				return errors.New("created_time not found")
			}

			// 需要等待至少1秒
			if curTime < createdTime+1 {
				fmt.Println("  [任务B1-异步检查] 冷却时间未到，继续等待...")
				return errors.New("waiting for cooldown period")
			}

			fmt.Println("  [任务B1-异步检查] 异步检查通过 ✓")
			return nil
		},
	))
	if err != nil {
		panic(err)
	}

	fmt.Println("✓ 所有任务节点注册成功")

	// 4. 创建工作流实例
	workflowInstance, err := workflowService.CreateWorkflow(context.Background(), &workflow.CreateWorkflowReq{
		WorkflowType: "complex_workflow",
		BusinessID:   "COMPLEX-001",
		Context: map[string]any{
			"created_time": time.Now().Unix(),
			"scenario":     "multi-branch-async",
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("✓ 工作流实例创建成功 (ID: %d)\n", workflowInstance.ID)

	// 5. 第一次运行
	fmt.Println("正在运行工作流（第一阶段）...")
	if err := workflowService.RunWorkflow(context.Background(), workflowInstance.ID); err != nil {
		panic(err)
	}

	// 6. 等待异步任务
	fmt.Println("等待异步任务冷却（2秒）...")
	time.Sleep(2 * time.Second)

	// 7. 第二次运行
	fmt.Println("继续运行工作流（完成异步任务）...")
	if err := workflowService.RunWorkflow(context.Background(), workflowInstance.ID); err != nil {
		panic(err)
	}

	fmt.Printf("✅ 复杂工作流执行完成！(实例 ID: %d)\n", workflowInstance.ID)
}

// runMultipleWorkflows 运行多个工作流实例示例
func runMultipleWorkflows(workflowService workflow.WorkflowService) {
	// 1. 定义简单工作流
	workflowConfigJson := `{
		"id": "simple_workflow",
		"name": "简单工作流",
		"nodes": [
			{
				"id": "step1",
				"name": "步骤1",
				"next_nodes": ["step2"]
			},
			{
				"id": "step2",
				"name": "步骤2",
				"next_nodes": ["step3"]
			},
			{
				"id": "step3",
				"name": "步骤3",
				"next_nodes": []
			}
		]
	}`

	workflowConfig := &workflow.WorkflowConfig{}
	if err := json.Unmarshal([]byte(workflowConfigJson), workflowConfig); err != nil {
		panic(err)
	}

	// 2. 加载配置
	if err := workflow.LoadWorkflowConfig(workflowConfig); err != nil {
		panic(err)
	}
	fmt.Println("✓ 简单工作流配置加载成功")

	// 3. 注册任务
	for i := 1; i <= 3; i++ {
		stepID := fmt.Sprintf("step%d", i)
		stepName := fmt.Sprintf("步骤%d", i)
		err := workflow.RegisterWorkflowTask("simple_workflow", stepID, workflow.NewNormalTaskWorker(
			func(ctx context.Context, nodeContext *workflow.JSONContext) error {
				nodeContext.Set([]string{"step"}, stepName)
				nodeContext.Set([]string{"timestamp"}, time.Now().Unix())
				return nil
			},
			nil,
		))
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("✓ 任务节点注册成功")

	// 4. 创建并运行多个工作流实例
	fmt.Println("正在创建并运行 5 个工作流实例...")

	instanceIDs := []int64{}
	for i := 0; i < 5; i++ {
		// 创建实例
		businessID := fmt.Sprintf("BATCH-%03d", i+1)
		instance, err := workflowService.CreateWorkflow(context.Background(), &workflow.CreateWorkflowReq{
			WorkflowType: "simple_workflow",
			BusinessID:   businessID,
			Context: map[string]any{
				"index":     i,
				"batch_id":  "BATCH-2024",
				"timestamp": time.Now().Unix(),
			},
		})
		if err != nil {
			panic(err)
		}

		// 运行实例
		if err := workflowService.RunWorkflow(context.Background(), instance.ID); err != nil {
			panic(err)
		}

		instanceIDs = append(instanceIDs, instance.ID)
		fmt.Printf("  ✓ 实例 %d: %s (ID: %d)\n", i+1, businessID, instance.ID)

		// 稍微延迟，避免时间戳完全相同
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("✅ 成功创建并执行 %d 个工作流实例！\n", len(instanceIDs))
	fmt.Printf("   实例 IDs: %v\n", instanceIDs)
}
