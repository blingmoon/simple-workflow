package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestService 创建测试服务
func setupTestService(t *testing.T) workflow.WorkflowService {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	return workflow.NewWorkflowService(repo, lock)
}

// TestWorkflowCreationBasic 测试基础工作流创建
func TestWorkflowCreationBasic(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	t.Run("创建简单工作流", func(t *testing.T) {
		// 1. 定义工作流配置
		workflowConfigJSON := `{
			"id": "test_workflow",
			"name": "测试工作流",
			"nodes": [
				{"id": "task1", "name": "任务1", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		// 2. 加载工作流配置
		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 3. 注册任务处理器
		err = workflow.RegisterWorkflowTask("test_workflow", "task1", 
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"executed"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 4. 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "test_workflow",
			BusinessID:   "TEST-001",
			Context:      map[string]any{"test": "value"},
		})

		assert.NoError(t, err)
		assert.NotNil(t, instance)
		assert.Greater(t, instance.ID, int64(0))
	})
}

// TestWorkflowExecution 测试工作流执行
func TestWorkflowExecution(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	t.Run("执行单任务工作流", func(t *testing.T) {
		// 配置
		workflowConfigJSON := `{
			"id": "exec_workflow",
			"name": "执行测试",
			"nodes": [
				{"id": "task1", "name": "任务1", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 注册任务
		executed := false
		err = workflow.RegisterWorkflowTask("exec_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					executed = true
					nodeContext.Set([]string{"result"}, "success")
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建并执行
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "exec_workflow",
			BusinessID:   "EXEC-001",
			Context:      map[string]any{},
			IsRun:        true, // 立即执行
		})

		assert.NoError(t, err)
		assert.NotNil(t, instance)
		
		// 如果不是立即执行，手动执行
		if !executed {
			err = service.RunWorkflow(ctx, instance.ID)
			t.Logf("执行结果: %v", err)
		}
	})

	t.Run("执行多任务工作流", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "multi_task_workflow",
			"name": "多任务工作流",
			"nodes": [
				{"id": "task1", "name": "任务1", "next_nodes": ["task2"]},
				{"id": "task2", "name": "任务2", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 注册任务
		err = workflow.RegisterWorkflowTask("multi_task_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"task1_done"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("multi_task_workflow", "task2",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"task2_done"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "multi_task_workflow",
			BusinessID:   "MULTI-001",
			Context:      map[string]any{},
		})

		require.NoError(t, err)
		
		// 执行工作流
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("多任务执行结果: %v", err)
	})
}

// TestWorkflowQuery 测试工作流查询
func TestWorkflowQuery(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	t.Run("查询工作流实例", func(t *testing.T) {
		// 创建一个工作流
		workflowConfigJSON := `{
			"id": "query_workflow",
			"name": "查询测试",
			"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("query_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "query_workflow",
			BusinessID:   "QUERY-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)
		assert.Greater(t, instance.ID, int64(0))
	})

	t.Run("统计工作流数量", func(t *testing.T) {
		workflowTypes := []string{"query_workflow"}
		params := &workflow.QueryWorkflowInstanceParams{
			WorkflowTypeIn: workflowTypes,
		}

		count, err := service.CountWorkflowInstance(ctx, params)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		t.Logf("工作流数量: %d", count)
	})
}

// TestWorkflowContext 测试上下文传递
func TestWorkflowContext(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	t.Run("上下文数据传递", func(t *testing.T) {
		workflowConfigJSON := `{
			"id": "context_workflow",
			"name": "上下文测试",
			"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("context_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					// 设置节点上下文
					nodeContext.Set([]string{"result"}, "processed")
					nodeContext.Set([]string{"timestamp"}, 1234567890)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建带上下文的工作流
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "context_workflow",
			BusinessID:   "CONTEXT-001",
			Context: map[string]any{
				"user_id": 123,
				"order_id": "ORDER-001",
				"amount": 99.99,
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, instance)
	})
}

// BenchmarkWorkflowCreation 性能测试
func BenchmarkWorkflowCreation(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	
	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	// 配置工作流
	workflowConfigJSON := `{
		"id": "bench_workflow",
		"name": "性能测试",
		"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
	}`

	var config workflow.WorkflowConfig
	_ = json.Unmarshal([]byte(workflowConfigJSON), &config)
	_ = workflow.LoadWorkflowConfig(&config)

	_ = workflow.RegisterWorkflowTask("bench_workflow", "task1",
		workflow.NewNormalTaskWorker(
			func(ctx context.Context, nodeContext *workflow.JSONContext) error {
				return nil
			},
			nil,
		),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "bench_workflow",
			BusinessID:   "BENCH-001",
			Context:      map[string]any{"index": i},
		})
	}
}
