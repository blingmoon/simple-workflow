package workflow

import (
	"context"

	"github.com/pkg/errors"
)

// WorkflowTaskNodeWorker 工作流任务节点工作器,需要外部实现
type WorkflowTaskNodeWorker interface {
	/**
	 * @description:  任务执行
	 * @return error nil表示执行成功了，不需要处理
	 * @param ctx context.Context 上下文
	 * @param nodeContext *JSONContext 节点上下文, run函数自己维护，run中更改了nodeContext,那么就会同步更改到数据库里面
	 */
	Run(ctx context.Context, nodeContext *JSONContext) error
	/**
	 * @description:  异步等待检查
	 * @return error nil表示检查成功了，不需要处理
	 * @param ctx context.Context 上下文
	 * @param nodeContext *JSONContext 节点上下文, run函数自己维护，run中更改了nodeContext,那么就会同步更改到数据库里面
	 */
	AsynchronousWaitCheck(ctx context.Context, nodeContext *JSONContext) error
}

var defaultEmptyTaskWorker WorkflowTaskNodeWorker = &EmptyTaskWorker{}

type EmptyTaskWorker struct {
}

func (w EmptyTaskWorker) Run(ctx context.Context, nodeContext *JSONContext) error {
	return errors.New("Not implemented")
}

func (w EmptyTaskWorker) AsynchronousWaitCheck(ctx context.Context, nodeContext *JSONContext) error {
	return errors.New("Not implemented")
}

type BaseTaskWorker struct {
	EmptyTaskWorker
}

/**
 * @description: 异步等待检查, base中不需要实现，需要实现，自己执行处理
 */
func (w BaseTaskWorker) AsynchronousWaitCheck(ctx context.Context, nodeContext *JSONContext) error {
	// 不是所有的节点都有异步等待检查
	return nil
}

/**
 * @description: 根节点工作器,工作流开始节点，特殊节点
 */
type RootNodeWorker struct {
	BaseTaskWorker
}

func (w RootNodeWorker) Run(ctx context.Context, nodeContext *JSONContext) error {
	return nil
}

/**
 * @description: 结束节点工作器,工作流结束节点,特殊节点
 */
type EndNodeWorker struct {
	BaseTaskWorker
}

func (w EndNodeWorker) Run(ctx context.Context, nodeContext *JSONContext) error {
	return nil
}

func NewRootTaskNodeDefinition() *WorkflowTaskNodeDefinition {
	return &WorkflowTaskNodeDefinition{
		TaskType:   rootTaskNode,
		TaskName:   "开始节点",
		PreNodes:   make([]*WorkflowTaskNodeDefinition, 0),
		NextNodes:  make([]*WorkflowTaskNodeDefinition, 0),
		TaskWorker: RootNodeWorker{},
	}
}

func NewEndTaskNodeDefinition() *WorkflowTaskNodeDefinition {
	return &WorkflowTaskNodeDefinition{
		TaskType:   endTaskNode,
		TaskName:   "结束节点",
		PreNodes:   make([]*WorkflowTaskNodeDefinition, 0),
		NextNodes:  make([]*WorkflowTaskNodeDefinition, 0),
		TaskWorker: &EndNodeWorker{},
	}
}

type RunFunc func(ctx context.Context, nodeContext *JSONContext) error
type AsynchronousWaitCheckFunc func(ctx context.Context, nodeContext *JSONContext) error
type NormalTaskWorker struct {
	BaseTaskWorker
	runHandler                   RunFunc
	asynchronousWaitCheckHandler AsynchronousWaitCheckFunc
}

func (w NormalTaskWorker) Run(ctx context.Context, nodeContext *JSONContext) error {
	if w.runHandler == nil {
		return w.BaseTaskWorker.Run(ctx, nodeContext)
	}
	return w.runHandler(ctx, nodeContext)
}

func (w NormalTaskWorker) AsynchronousWaitCheck(ctx context.Context, nodeContext *JSONContext) error {
	if w.asynchronousWaitCheckHandler == nil {
		return w.BaseTaskWorker.AsynchronousWaitCheck(ctx, nodeContext)
	}
	return w.asynchronousWaitCheckHandler(ctx, nodeContext)
}

func NewNormalTaskWorker(
	funcRun RunFunc,
	funcAsynchronousWaitCheck AsynchronousWaitCheckFunc,
) *NormalTaskWorker {
	return &NormalTaskWorker{
		BaseTaskWorker:               BaseTaskWorker{},
		runHandler:                   funcRun,
		asynchronousWaitCheckHandler: funcAsynchronousWaitCheck,
	}
}
