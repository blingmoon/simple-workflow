package workflow

import "context"

type WorkflowService interface {
	/**
	 * @description: 创建工作流
	 * @param ctx context.Context
	 * @param req *CreateWorkflowReq
	 * @return *WorkflowInstance, error
	 */
	CreateWorkflow(ctx context.Context, req *CreateWorkflowReq) (*WorkflowInstance, error)
	/**
	 * @description: 查询工作流实例数量
	 * @param ctx context.Context
	 * @param params *QueryWorkflowInstanceParams
	 * @return int64, error
	 */
	CountWorkflowInstance(ctx context.Context, params *QueryWorkflowInstanceParams) (int64, error)
	/**
	 * @description: 查询工作流实例详情
	QueryWorkflowInstanceDetail(ctx context.Context, params *QueryWorkflowInstanceParams) ([]*WorkflowInstanceDetailEntity, error)
	QueryWorkflowInstancePo(ctx context.Context, params *QueryWorkflowInstanceParams) ([]*WorkflowInstancePo, error)
	/**
	 * @description: 运行工作流
	 *				 workflowID 为工作流实例ID,一个工作流实例只会被一个goroutine运行
	 *				 如果有其他goroutine正在运行该工作流实例，则返回错误
	 * @param ctx context.Context
	 * @param workflowID int64
	 * @return error
	*/
	RunWorkflow(ctx context.Context, workflowID int64) error
	/**
	 * @description: 取消工作流，手动取消工作流，目前没有使用，将来使用扩展
	 *				 workflowInstanceID 为工作流实例ID,一个工作流实例只会被一个goroutine运行
	 *				 如果有其他goroutine正在运行该工作流实例，则返回错误
	 * @param ctx context.Context
	 * @param workflowInstanceID int64
	 * @return error
	 */
	CancelWorkflowInstance(ctx context.Context, workflowInstanceID int64) error

	/**
	 * @description: 添加节点外部事件, 给外部输入使用，会写入节点上下文
	 * 				  一个工作流实例只会被一个goroutine运行,如果有其他goroutine正在运行该工作流实例，则返回错误
	 *                会将事件addParams.NodeEvent写入额外节点上下文的 NodeContextKeyNodeEvent(node_event)
	 * @param ctx context.Context
	 * @param addParams *AddNodeExternalEventParams
	 *				  addParams.WorkflowInstanceID 为工作流实例ID
	 *				  addParams.TaskType 为任务类型
	 *				  addParams.NodeEvent 为节点事件
	 *				  addParams.NodeEvent.EventTs 为事件时间, 会用来做版本控制，最新的版本会覆盖旧的版本
	 *				  addParams.NodeEvent.EventContent 为事件内容
	 * @return error
	 */
	AddNodeExternalEvent(ctx context.Context, addParams *AddNodeExternalEventParams) error

	/**
	 * @description: 重新开始工作流任务
	 *                如果isAsynchronous为true,一次性只能有一个进程操作工作流,可能会出现当前工作流正在被其他进程操作的情况
	 *                所以失败的情况可能会出现的比较频繁
	 * @param ctx context.Context
	 * @param restartParams *RestartWorkflowTaskParams
	 *				  restartWorkflowNodeParams.WorkflowInstanceID 为工作流实例ID
	 *				  restartWorkflowNodeParams.TaskType 为任务类型
	 *				  restartWorkflowNodeParams.IsAsynchronous 为是否异步重启,如果为true，则不等待任务执行完成,直接返回
	 * @return error
	 */
	RestartWorkflowNode(ctx context.Context, restartWorkflowNodeParams *RestartWorkflowNodeParams) error

	/**
	 * @description: 重启工作流实例, 只有失败和取消状态可以重启，正常完成的不能重启
	 * @param ctx context.Context
	 * @param restartWorkflowParams *RestartWorkflowParams 重启工作流参数
	 *				  restartWorkflowParams.WorkflowInstanceID 为工作流实例ID
	 *				  restartWorkflowParams.Context 为上下文,如果有值，则覆盖掉原来的上下文
	 *				  restartWorkflowParams.IsRun 为是否立即执行,如果为true，则立即执行
	 * @return error 重启工作流实例
	 */
	RestartWorkflowInstance(ctx context.Context, restartWorkflowParams *RestartWorkflowParams) error
}

// WorkflowServiceImpl 工作流服务
type WorkflowServiceImpl struct {
	repo        WorkflowRepo
	executeLock WorkflowLock
}

func NewWorkflowService(repo WorkflowRepo, executeLock WorkflowLock) WorkflowService {
	return &WorkflowServiceImpl{repo: repo, executeLock: executeLock}
}
