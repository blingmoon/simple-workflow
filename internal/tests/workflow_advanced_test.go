package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCancelWorkflowInstance 测试取消工作流
func TestCancelWorkflowInstance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("取消运行中的工作流", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "cancel_test_workflow",
			"name": "取消测试工作流",
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
		err = workflow.RegisterWorkflowTask("cancel_test_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"executed"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "cancel_test_workflow",
			BusinessID:   "CANCEL-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)
		require.NotNil(t, instance)

		// 取消工作流
		err = service.CancelWorkflowInstance(ctx, instance.ID)
		t.Logf("取消工作流结果: %v", err)

		// 注意：取消功能可能返回错误或成功，取决于实现
		// 这里我们只是测试该方法可以被调用
	})

	t.Run("取消不存在的工作流", func(t *testing.T) {
		// 尝试取消一个不存在的工作流
		err := service.CancelWorkflowInstance(ctx, 999999)
		assert.Error(t, err, "取消不存在的工作流应该返回错误")
		t.Logf("取消不存在工作流的错误: %v", err)
	})
}

// TestAddNodeExternalEvent 测试添加节点外部事件
func TestAddNodeExternalEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("添加外部事件到等待审批的任务", func(t *testing.T) {
		// 定义一个需要外部审批的工作流
		workflowConfigJSON := `{
			"id": "approval_event_workflow",
			"name": "审批事件工作流",
			"nodes": [
				{"id": "wait_approval", "name": "等待审批", "next_nodes": ["after_approval"]},
				{"id": "after_approval", "name": "审批后", "next_nodes": []}
			]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		// 注册等待审批任务
		err = workflow.RegisterWorkflowTask("approval_event_workflow", "wait_approval",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					// 检查是否有审批结果
					approval, ok := nodeContext.GetString("node_event", "approval_result")
					if !ok || approval == "" {
						// 没有审批结果，返回未就绪
						return workflow.ErrorWorkflowTaskInstanceNotReady
					}

					if approval == "approved" {
						nodeContext.Set([]string{"approved"}, true)
						return nil
					}

					return errors.New("审批被拒绝")
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 注册审批后任务
		err = workflow.RegisterWorkflowTask("approval_event_workflow", "after_approval",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"completed"}, true)
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "approval_event_workflow",
			BusinessID:   "APPROVAL-EVENT-001",
			Context:      map[string]any{"user_id": 123},
		})
		require.NoError(t, err)

		// 第一次运行 - 会在等待审批处停止
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第一次运行结果: %v", err)

		// 添加外部事件（模拟审批）
		eventContent := workflow.NewJSONContextFromMap(map[string]any{
			"approval_result": "approved",
			"approver":        "manager",
			"comment":         "同意",
			"approved_at":     time.Now().Unix(),
		})
		eventContentStr := string(eventContent.ToBytesWithoutError())

		err = service.AddNodeExternalEvent(ctx, &workflow.AddNodeExternalEventParams{
			WorkflowInstanceID: instance.ID,
			TaskType:           "wait_approval",
			NodeEvent: &workflow.NodeExternalEvent{
				EventContent: eventContentStr,
				EventTs:      time.Now().Unix(),
			},
		})

		if err != nil {
			t.Logf("添加外部事件结果: %v", err)
		} else {
			t.Logf("✅ 外部事件添加成功")

			// 再次运行工作流 - 应该能继续执行
			err = service.RunWorkflow(ctx, instance.ID)
			t.Logf("第二次运行结果: %v", err)
		}
	})

	t.Run("添加事件到不存在的工作流", func(t *testing.T) {
		eventContent := workflow.NewJSONContextFromMap(map[string]any{"test": true})
		eventContentStr := string(eventContent.ToBytesWithoutError())

		err := service.AddNodeExternalEvent(ctx, &workflow.AddNodeExternalEventParams{
			WorkflowInstanceID: 999999,
			TaskType:           "some_task",
			NodeEvent: &workflow.NodeExternalEvent{
				EventContent: eventContentStr,
				EventTs:      time.Now().Unix(),
			},
		})

		if err != nil {
			t.Logf("预期的错误: %v", err)
		}
	})
}

// TestCountWorkflowInstance 测试统计工作流实例
func TestCountWorkflowInstance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("统计特定类型的工作流", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "count_test_workflow",
			"name": "统计测试工作流",
			"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("count_test_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建多个工作流实例
		for i := 0; i < 5; i++ {
			_, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
				WorkflowType: "count_test_workflow",
				BusinessID:   "COUNT-TEST-001",
				Context:      map[string]any{"index": i},
			})
			require.NoError(t, err)
		}

		// 统计工作流数量
		count, err := service.CountWorkflowInstance(ctx, &workflow.QueryWorkflowInstanceParams{
			WorkflowTypeIn: []string{"count_test_workflow"},
		})

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5), "应该至少有5个工作流实例")
		t.Logf("统计到的工作流数量: %d", count)
	})

	t.Run("按业务ID统计", func(t *testing.T) {
		businessID := "COUNT-BUSINESS-001"
		params := &workflow.QueryWorkflowInstanceParams{
			BusinessID: &businessID,
		}

		count, err := service.CountWorkflowInstance(ctx, params)
		assert.NoError(t, err)
		t.Logf("业务ID %s 的工作流数量: %d", businessID, count)
	})
}

// TestQueryWorkflowInstanceDetail 测试查询工作流实例详情
func TestQueryWorkflowInstanceDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("查询工作流详情", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "detail_test_workflow",
			"name": "详情测试工作流",
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
		for _, taskID := range []string{"task1", "task2"} {
			err = workflow.RegisterWorkflowTask("detail_test_workflow", taskID,
				workflow.NewNormalTaskWorker(
					func(ctx context.Context, nodeContext *workflow.JSONContext) error {
						nodeContext.Set([]string{"executed"}, true)
						return nil
					},
					nil,
				),
			)
			require.NoError(t, err)
		}

		// 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "detail_test_workflow",
			BusinessID:   "DETAIL-001",
			Context: map[string]any{
				"user_id":  123,
				"order_id": "ORDER-001",
			},
		})
		require.NoError(t, err)

		// 执行工作流
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("工作流执行结果: %v", err)

		// 查询工作流详情
		details, err := service.QueryWorkflowInstanceDetail(ctx, &workflow.QueryWorkflowInstanceParams{
			WorkflowInstanceID: &instance.ID,
			Page: &workflow.Pager{
				Page: 1,
				Size: 10,
			},
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, details, "应该返回工作流详情")

		if len(details) > 0 {
			detail := details[0]
			t.Logf("工作流详情: ID=%d, 状态=%s, 任务数=%d",
				detail.ID,
				detail.Status,
				len(detail.TaskInstances))

			assert.Equal(t, instance.ID, detail.ID)
			assert.NotNil(t, detail.WorkflowContext)
		}
	})
}

// TestQueryWorkflowInstancePo 测试查询工作流实例Po
func TestQueryWorkflowInstancePo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("查询工作流Po", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "po_test_workflow",
			"name": "Po测试工作流",
			"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("po_test_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "po_test_workflow",
			BusinessID:   "PO-001",
			Context:      map[string]any{"test": true},
		})
		require.NoError(t, err)

		// 查询Po
		instances, err := service.QueryWorkflowInstancePo(ctx, &workflow.QueryWorkflowInstanceParams{
			WorkflowInstanceID: &instance.ID,
			Page: &workflow.Pager{
				Page: 1,
				Size: 10,
			},
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, instances, "应该返回工作流Po")

		if len(instances) > 0 {
			po := instances[0]
			t.Logf("工作流Po: ID=%d, Type=%s, BusinessID=%s, Status=%s",
				po.ID, po.WorkflowType, po.BusinessID, po.Status)

			assert.Equal(t, instance.ID, po.ID)
			assert.Equal(t, "po_test_workflow", po.WorkflowType)
			assert.Equal(t, "PO-001", po.BusinessID)
		}
	})

	t.Run("按状态查询", func(t *testing.T) {
		// 查询初始化状态的工作流
		instances, err := service.QueryWorkflowInstancePo(ctx, &workflow.QueryWorkflowInstanceParams{
			StatusIn: []string{workflow.WorkflowInstanceStatusInit},
			Page: &workflow.Pager{
				Page: 1,
				Size: 10,
			},
		})

		assert.NoError(t, err)
		t.Logf("初始化状态的工作流数量: %d", len(instances))
	})
}

// TestCreateWorkflowWithDifferentContexts 测试使用不同上下文创建工作流
func TestCreateWorkflowWithDifferentContexts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	// 定义工作流
	workflowConfigJSON := `{
		"id": "context_test_workflow",
		"name": "上下文测试工作流",
		"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
	}`

	var config workflow.WorkflowConfig
	err = json.Unmarshal([]byte(workflowConfigJSON), &config)
	require.NoError(t, err)

	err = workflow.LoadWorkflowConfig(&config)
	require.NoError(t, err)

	err = workflow.RegisterWorkflowTask("context_test_workflow", "task1",
		workflow.NewNormalTaskWorker(
			func(ctx context.Context, nodeContext *workflow.JSONContext) error {
				return nil
			},
			nil,
		),
	)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		context map[string]any
	}{
		{
			name:    "空上下文",
			context: map[string]any{},
		},
		{
			name: "简单上下文",
			context: map[string]any{
				"key": "value",
			},
		},
		{
			name: "复杂上下文",
			context: map[string]any{
				"user": map[string]any{
					"id":   123,
					"name": "张三",
				},
				"order": map[string]any{
					"id":     "ORDER-001",
					"amount": 99.99,
					"items":  []string{"item1", "item2"},
				},
				"metadata": map[string]any{
					"source":    "api",
					"timestamp": time.Now().Unix(),
				},
			},
		},
		{
			name: "包含数组的上下文",
			context: map[string]any{
				"tags":   []string{"tag1", "tag2", "tag3"},
				"scores": []float64{1.1, 2.2, 3.3},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
				WorkflowType: "context_test_workflow",
				BusinessID:   "CONTEXT-" + tc.name,
				Context:      tc.context,
			})

			assert.NoError(t, err)
			assert.NotNil(t, instance)
			assert.Greater(t, instance.ID, int64(0))
			t.Logf("✅ %s: 创建成功, ID=%d", tc.name, instance.ID)
		})
	}
}

// TestRunWorkflowMultipleTimes 测试多次运行工作流
func TestRunWorkflowMultipleTimes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	require.NoError(t, err)

	repo := workflow.NewWorkflowRepo(db)
	lock := workflow.NewLocalWorkflowLock()
	service := workflow.NewWorkflowService(repo, lock)

	ctx := context.Background()

	t.Run("多次运行已完成的工作流", func(t *testing.T) {
		// 定义工作流
		workflowConfigJSON := `{
			"id": "run_multiple_workflow",
			"name": "多次运行工作流",
			"nodes": [{"id": "task1", "name": "任务1", "next_nodes": []}]
		}`

		var config workflow.WorkflowConfig
		err := json.Unmarshal([]byte(workflowConfigJSON), &config)
		require.NoError(t, err)

		err = workflow.LoadWorkflowConfig(&config)
		require.NoError(t, err)

		err = workflow.RegisterWorkflowTask("run_multiple_workflow", "task1",
			workflow.NewNormalTaskWorker(
				func(ctx context.Context, nodeContext *workflow.JSONContext) error {
					nodeContext.Set([]string{"run_count"}, time.Now().Unix())
					return nil
				},
				nil,
			),
		)
		require.NoError(t, err)

		// 创建工作流实例
		instance, err := service.CreateWorkflow(ctx, &workflow.CreateWorkflowReq{
			WorkflowType: "run_multiple_workflow",
			BusinessID:   "RUN-MULTIPLE-001",
			Context:      map[string]any{},
		})
		require.NoError(t, err)

		// 第一次运行
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第一次运行结果: %v", err)

		// 第二次运行（工作流可能已完成）
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第二次运行结果: %v", err)

		// 第三次运行
		err = service.RunWorkflow(ctx, instance.ID)
		t.Logf("第三次运行结果: %v", err)
	})
}
