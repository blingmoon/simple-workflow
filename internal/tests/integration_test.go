package tests

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test完整工作流场景
func TestCompleteWorkflowScenario(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("订单处理工作流", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "order_workflow",
			"name": "订单处理",
			"nodes": [
				{"id": "validate", "name": "验证", "next_nodes": ["pay"]},
				{"id": "pay", "name": "支付", "next_nodes": ["ship"]},
				{"id": "ship", "name": "发货", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 注册任务
		err = workflow.RegisterWorkflowTask("order_workflow", "validate",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"validated"}, true)
					nodeContext.Set([]string{"validate_time"}, time.Now().Unix())
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("order_workflow", "pay",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"paid"}, true)
					nodeContext.Set([]string{"transaction_id"}, "TXN-123")
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("order_workflow", "ship",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"shipped"}, true)
					nodeContext.Set([]string{"tracking_number"}, "TRACK-789")
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "order_workflow",
			BusinessID:   "ORDER-001",
			Context: map[string]any{
				"user_id": 123,
				"amount":  99.99,
			},
		})
		require.NoError(t, err)

		// 执行
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("订单工作流执行结果: %v", err)
	})
}

// TestAsyncWorkflow 测试异步工作流
func TestAsyncWorkflow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("异步检查任务", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "async_workflow",
			"name": "异步工作流",
			"nodes": [
				{"id": "async_task", "name": "异步任务", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 注册带异步检查的任务
		asyncStartTime := time.Now().Unix()
		err = workflow.RegisterWorkflowTask("async_workflow", "async_task",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"started"}, true)
					nodeContext.Set([]string{"start_time"}, asyncStartTime)
					return nil
				},
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					// 异步检查：等待5秒
					startTime, ok := nodeContext.GetInt64("start_time")
					if !ok {
						return errors.New("start_time not found")
					}
					
					elapsed := time.Now().Unix() - startTime
					if elapsed < 2 {
						// 还没到时间
						return workflow.ErrorWorkflowTaskInstanceNotReady
					}
					
					// 时间到了，标记完成
					nodeContext.Set([]string{"async_checked"}, true)
					return nil
				},
			),
		)
		require.NoError(t, err)

		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "async_workflow",
			BusinessID:   "ASYNC-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)

		// 第一次执行 - 可能会返回 NotReady
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第一次执行: %v", err)

		// 等待一下
		time.Sleep(2 * time.Second)

		// 第二次执行 - 应该完成
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第二次执行: %v", err)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("任务执行失败", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "error_workflow",
			"name": "错误工作流",
			"nodes": [
				{"id": "fail_task", "name": "失败任务", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("error_workflow", "fail_task",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					return errors.New("任务执行失败")
				},
				nil,
			),
		)
		require.NoError(t, err)

		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "error_workflow",
			BusinessID:   "ERROR-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)

		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("预期的失败: %v", err)
	})

	t.Run("特殊错误-继续执行", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "continue_workflow",
			"name": "继续执行工作流",
			"nodes": [
				{"id": "optional_task", "name": "可选任务", "next_nodes": ["next_task"]},
				{"id": "next_task", "name": "下一个任务", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("continue_workflow", "optional_task",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					// 返回可继续的错误
					return workflow.ErrorWorkflowTaskFailedWithContinue
				},
				nil,
			),
		)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("continue_workflow", "next_task",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"reached_next"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "continue_workflow",
			BusinessID:   "CONTINUE-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)

		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("继续执行结果: %v", err)
	})
}

// TestConcurrentExecution 测试并发执行
func TestConcurrentExecution(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("并发创建工作流", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "concurrent_workflow",
			"name": "并发工作流",
			"nodes": [
				{"id": "task", "name": "任务", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("concurrent_workflow", "task",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"executed"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 并发创建
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				_, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
					WorkflowType: "concurrent_workflow",
					BusinessID:   "CONCURRENT-" + string(rune(index)),
					Context:      map[string]any{"index": index},
				})

				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		t.Logf("并发创建成功: %d/10", successCount)
		assert.Greater(t, successCount, 0)
	})
}
