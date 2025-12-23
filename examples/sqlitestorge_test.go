package examples

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSqliteStore(t *testing.T) {
	workflowService, err := NewSqliteStore()
	if err != nil {
		t.Fatalf("NewSqliteStore failed, err: %v", err)
	}
	// 声明工作流配置
	// A->B->C
	// B->D-C
	workflowConfigJson := `
{
		"id": "test",
		"name": "test",
		"nodes": [
			{
				"id": "A",
				"name": "A",
				"next_nodes": ["B"]
			},
			{
				"id": "A1",
				"name": "A1",
				"next_nodes": ["B","B1"]
			},
			{
				"id": "B",
				"name": "B",
				"next_nodes": ["C"]
			},
			{
				"id": "B1",
				"name": "B1",
				"next_nodes": []
			},
			{
				"id": "C",
				"name": "C",
				"next_nodes": []
			}
		]
	}
	`
	workflowConfig := &workflow.WorkflowConfig{}
	err = json.Unmarshal([]byte(workflowConfigJson), workflowConfig)
	require.Nil(t, err)
	err = workflow.LoadWorkflowConfig(workflowConfig)
	require.Nil(t, err)
	//注册工作流任务节点
	err = workflow.RegisterWorkflowTask("test", "A", workflow.NewNormalTaskWorker(func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		nodeContext.Set([]string{"name"}, "A")
		return nil
	}, func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		return nil
	}))
	require.Nil(t, err)
	err = workflow.RegisterWorkflowTask("test", "A1", workflow.NewNormalTaskWorker(func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		nodeContext.Set([]string{"name"}, "A1")
		return nil
	}, nil))
	require.Nil(t, err)
	err = workflow.RegisterWorkflowTask("test", "B", workflow.NewNormalTaskWorker(func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		nodeContext.Set([]string{"name"}, "B")
		return nil
	}, nil))
	require.Nil(t, err)
	err = workflow.RegisterWorkflowTask("test", "B1", workflow.NewNormalTaskWorker(func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		nodeContext.Set([]string{"name"}, "B1")
		return nil
	}, func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		curTime := time.Now().Unix()
		createdTime, ok := nodeContext.GetInt64("workflow_context", "created_time")
		if !ok {
			return errors.New("created_time is not found")
		}
		if curTime < createdTime+1 {
			// 等待1s
			return errors.New("created_time is not equal to curTime")
		}
		return nil
	}))
	require.Nil(t, err)
	err = workflow.RegisterWorkflowTask("test", "C", workflow.NewNormalTaskWorker(func(ctx context.Context, nodeContext *workflow.JSONContext) error {
		nodeContext.Set([]string{"name"}, "C")
		return nil
	}, nil))
	require.Nil(t, err)

	workflowInstance, err := workflowService.CreateWorkflow(context.Background(), &workflow.CreateWorkflowReq{
		WorkflowType: "test",
		BusinessID:   "test",
		Context:      map[string]any{"created_time": time.Now().Unix()},
	})
	require.Nil(t, err)
	require.NotNil(t, workflowInstance)
	err = workflowService.RunWorkflow(context.Background(), workflowInstance.ID)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
	err = workflowService.RunWorkflow(context.Background(), workflowInstance.ID)
	require.Nil(t, err)

}
