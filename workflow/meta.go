package workflow

import "github.com/pkg/errors"

var (
	ErrWorkflowConfigNotFound              = errors.New("workflow config not found")
	ErrWorkflowDefinitionNotFound          = errors.New("workflow definition not found")
	ErrWorkflowTaskWorkerNotFound          = errors.New("workflow task worker not found")
	ErrWorkflowTaskWorkerAlreadyRegistered = errors.New("workflow task worker already registered")
	ErrWorkflowInstanceNotFound            = errors.New("workflow instance not found")
	ErrWorkflowTaskInstanceNotFound        = errors.New("workflow task instance not found")
	// 特殊的error 会影响流程的error
	//ErrorWorkflowTaskInstanceNotReady: 当前阶段还没有准备好，需要过一会儿来重试
	// 场景&应用: 审核中，每次都是审核中
	ErrorWorkflowTaskInstanceNotReady = errors.New("workflow task instance not ready") // 任务实例没有准备好，是正常的错误，常见的
	// ErrorWorkflowTaskFailedWithContinue: 任务实例失败，但是可以继续执行，可能当成另外一种完成
	// 场景&应用: 一些异步任务，对这个任务结果不关系，成功或者失败都行
	ErrorWorkflowTaskFailedWithContinue = errors.New("workflow task failed with continue") // 任务实例失败，但是可以继续执行
	// ErrWorkflowTaskFailedWithFailed: 任务实例失败，这个任务失败，整个工作流状态变成failed
	// 场景&应用: 一些关键参数丢失，整个工作流需要终止，无论重试多少次都不会成功
	ErrWorkflowTaskFailedWithFailed = errors.New("workflow task failed with termination") // 任务实例失败，但是终止整个工作流，工作流状态变成failed

	// 下面这个两个错误信息给业务上面使用,目前用于报警定义
	// 如果你希望这种错误在定时脚本打印error 使用errors.Wrapf(ErrWorkBussinessCriticalError, "err message: %s", err)
	// 如果你希望这种错误在定时脚本打印warn 使用errors.Wrapf(ErrWorkBussinessWarningError, "err message: %s", err)
	ErrWorkBussinessCriticalError = errors.New("work bussiness critical error") // 业务严重错误，需要人工介入处理
	ErrWorkBussinessWarningError  = errors.New("work bussiness warning error")  // 业务警告错误，答应warn级别日志
)

var (
	rootTaskNode = "root"
	endTaskNode  = "end"
)

type WorkflowInstanceStatus = string

const (
	WorkflowInstanceStatusInit    WorkflowInstanceStatus = "init"
	WorkflowInstanceStatusRunning WorkflowInstanceStatus = "running"
	// 完成, 工作流终止状态, 不再重试 普遍含义: 任务执行成功
	WorkflowInstanceStatusCompleted WorkflowInstanceStatus = "completed"
	// 失败, 工作流终止状态, 不再重试 普遍含义: 任务执行失败, 工作流终止,某个节点原因导致工作流终止
	WorkflowInstanceStatusFailed WorkflowInstanceStatus = "failed"
	// 取消, 工作流终止状态, 不再重试 普遍含义: 任务执行取消, 手动取消的，和人工手动操作有关系，目前没有使用到
	WorkflowInstanceStatusCancelled WorkflowInstanceStatus = "canceled"
)

func IsOverWorkflowInstanceStatus(status WorkflowInstanceStatus) bool {
	return status == WorkflowInstanceStatusFailed || status == WorkflowInstanceStatusCancelled || status == WorkflowInstanceStatusCompleted
}

func GetWorkflowInstanceStatusText(status WorkflowInstanceStatus) string {
	switch status {
	case WorkflowInstanceStatusInit:
		return "初始化"
	case WorkflowInstanceStatusRunning:
		return "运行中"
	case WorkflowInstanceStatusCompleted:
		return "完成"
	case WorkflowInstanceStatusFailed:
		return "失败"
	case WorkflowInstanceStatusCancelled:
		return "取消"
	}
	return "未知"
}

type WorkflowTaskNodeStatus = string

const (
	WorkflowTaskNodeStatusStatusUnCreated WorkflowInstanceStatus = "uncreated"  // 数据库中不存在这个状态,用于标识工作流任务实例任务未创建
	WorkflowTaskNodeStatusInit            WorkflowTaskNodeStatus = "init"       // 初始化状态,数据库里面没有这种状态
	WorkflowTaskNodeStatusRestarting      WorkflowTaskNodeStatus = "restarting" // 重新启动状态,状态上和init是是一样的，但是初始化工作已经完成了
	WorkflowTaskNodeStatusRunning         WorkflowTaskNodeStatus = "running"
	WorkflowTaskNodeStatusPending         WorkflowTaskNodeStatus = "pending"
	WorkflowTaskNodeStatusFinishing       WorkflowTaskNodeStatus = "finishing"
	// 完成, 工作流终止状态, 不再重试 普遍含义: 任务执行成功, 注意针对ErrorWorkflowTaskFailedWithContinue这个错误,任务节点是completed而不是failed
	WorkflowTaskNodeStatusCompleted WorkflowTaskNodeStatus = "completed"
	// 失败, 工作流终止状态, 不再重试 普遍含义: 任务执行失败, 工作流终止,某个节点原因导致工作流终止
	WorkflowTaskNodeStatusFailed WorkflowTaskNodeStatus = "failed"
	// 取消, 工作流终止状态, 不再重试 普遍含义: 任务执行取消, 手动取消的，和人工手动操作有关系，目前没有使用到
	WorkflowTaskNodeStatusCancelled WorkflowTaskNodeStatus = "canceled"
)

func IsOverWorkflowTaskNodeStatus(status WorkflowTaskNodeStatus) bool {
	return status == WorkflowTaskNodeStatusFailed || status == WorkflowTaskNodeStatusCancelled || status == WorkflowTaskNodeStatusCompleted
}

// NodeContextKey 节点上下文key,用于获取节点上下文中的值
type NodeContextKey = string

const (
	NodeContextKeyNodeEvent       NodeContextKey = "node_event"
	NodeContextKeySystem          NodeContextKey = "system"
	NodeContextKeyPreNodeContext  NodeContextKey = "pre_node_context"
	NodeContextKeyWorkflowContext NodeContextKey = "workflow_context"
	// 备注原因，一般和节点失败相关，表明为什么失败
	NodeContextKeyReason NodeContextKey = "reason"
)

func GetWorkflowTaskNodeStatusText(status WorkflowTaskNodeStatus) string {
	switch status {
	case WorkflowTaskNodeStatusStatusUnCreated:
		return "未创建"
	case WorkflowTaskNodeStatusInit:
		return "初始化"
	case WorkflowTaskNodeStatusRestarting:
		return "重新启动"
	case WorkflowTaskNodeStatusRunning:
		return "运行中"
	case WorkflowTaskNodeStatusPending:
		return "等待中"
	case WorkflowTaskNodeStatusFinishing:
		return "完成中"
	case WorkflowTaskNodeStatusCancelled:
		return "取消"
	case WorkflowTaskNodeStatusFailed:
		return "失败"
	case WorkflowTaskNodeStatusCompleted:
		return "完成"
	}
	return "未知"
}

// IsSeriousError 目前只用在workflow_trigger_cron.go中，
// 用于判断是否是严重错误，如果是严重错误，则打error级别日志，
// 否则打warn级别日志
// 严重错误定义：需要人工介入处理处理，
// 1. 当前工作流实例不会重试，异常结束，如果一些绑定异常的数据
// 2. 或者当前工作流实例没有办法正常运行，需要人工介入处理,如配置不正确
func IsSeriousError(err error) bool {
	if err == nil {
		// 空error不算严重错误
		return false
	}
	causeErr := errors.Cause(err)
	if errors.Is(causeErr, ErrWorkflowConfigNotFound) ||
		errors.Is(causeErr, ErrWorkflowDefinitionNotFound) ||
		errors.Is(causeErr, ErrWorkflowTaskWorkerNotFound) ||
		errors.Is(causeErr, ErrWorkflowTaskWorkerAlreadyRegistered) ||
		errors.Is(causeErr, ErrWorkflowInstanceNotFound) ||
		errors.Is(causeErr, ErrWorkflowTaskInstanceNotFound) ||
		errors.Is(causeErr, ErrWorkflowTaskFailedWithFailed) ||
		errors.Is(causeErr, ErrorWorkflowTaskFailedWithContinue) ||
		errors.Is(causeErr, ErrWorkBussinessCriticalError) {
		return true
	}
	return false

}
