package workflow

import (
	"context"
)

type WorkflowRepo interface {
	CreateWorkflowInstance(ctx context.Context, workflowInstance *WorkflowInstancePo) (*WorkflowInstancePo, error)
	CreateWorkflowTaskInstance(ctx context.Context, workflowTaskInstance *WorkflowTaskInstancePo) (*WorkflowTaskInstancePo, error)
	QueryWorkflowInstance(ctx context.Context, param *QueryWorkflowInstanceParams) ([]*WorkflowInstancePo, error)
	CountWorkflowInstance(ctx context.Context, param *QueryWorkflowInstanceParams) (int64, error)
	QueryWorkflowTaskInstance(ctx context.Context, param *QueryWorkflowTaskInstanceParams) ([]*WorkflowTaskInstancePo, error)
	UpdateWorkflowInstance(ctx context.Context, param *UpdateWorkflowInstanceParams) error
	UpdateWorkflowTaskInstance(ctx context.Context, param *UpdateWorkflowTaskInstanceParams) error
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
