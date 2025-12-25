package workflow

import (
	"context"
	goerrors "errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// 辅助函数：替代 String 和 Bool
func String(s string) *string { return &s }
func Bool(b bool) *bool       { return &b }

var (
	workflowTaskWorkers = sync.Map{}
	workflowConfigs     = sync.Map{}
	workflowDefinitions = sync.Map{}
	loadWorkflowLock    = sync.Mutex{}
)

// WorkflowTaskNode 工作流任务节点entity
type WorkflowTaskNode struct {
	ID                 int64
	WorkflowInstanceID int64
	TaskType           string
	Status             string
	NodeContext        *JSONContext
	CreatedAt          int64
	UpdatedAt          int64
	FailCount          int64
}

// WorkflowDefinition 工作流定义entity
type WorkflowDefinition struct {
	ID         string
	Name       string
	NodesCount int64
	RootNode   *WorkflowTaskNodeDefinition
	Nodes      []*WorkflowTaskNodeDefinition // 节点列表,冗余字段,方便构建节点详情
}

// WorkflowTaskNodeDefinition 工作流任务节点定义entity
type WorkflowTaskNodeDefinition struct {
	TaskType      string
	TaskName      string
	FailMaxCount  int64 // 失败次数达到 fail_max_count次后，<0的忽略
	MaxWaitTimeTs int64 // 最大等待时间，单位秒，<=0 忽略
	PreNodes      []*WorkflowTaskNodeDefinition
	NextNodes     []*WorkflowTaskNodeDefinition
	TaskWorker    WorkflowTaskNodeWorker // 工作流任务节点工作器,需要外部实现
}

type CreateWorkflowReq struct {
	WorkflowType string         // 工作流类型
	BusinessID   string         // 业务ID
	Context      map[string]any // 上下文,可以为空
	IsRun        bool           // 是否立即执行,如果为true，则立即执行
	TaskId       int64          // 任务id
}

func getWorkflowTaskWorker(workflowType string, taskKey string) (WorkflowTaskNodeWorker, bool) {
	worker, ok := workflowTaskWorkers.Load(workflowType + "_" + taskKey)
	if !ok {
		return defaultEmptyTaskWorker, false
	}
	workerHandler, ok := worker.(WorkflowTaskNodeWorker)
	if !ok {
		return defaultEmptyTaskWorker, false
	}
	return workerHandler, true
}

func getAllChildrenTaskType(node *WorkflowTaskNodeDefinition) []string {
	if len(node.NextNodes) == 0 {
		// 没有后置节点，直接返回
		return []string{}
	}
	ret := make([]string, 0)
	for _, nextNode := range node.NextNodes {
		ret = append(ret, nextNode.TaskType)
		ret = append(ret, getAllChildrenTaskType(nextNode)...)
	}
	ret = UniqueStr(ret)
	return ret
}
func UniqueStr(arr []string) []string {
	ret := make([]string, 0)
	arrItemMap := make(map[string]struct{})
	for _, v := range arr {
		if _, ok := arrItemMap[v]; !ok {
			ret = append(ret, v)
			arrItemMap[v] = struct{}{}
		}
	}
	return ret
}

// WorkflowConfig 工作流配置,流程配置
type WorkflowConfig struct {
	ID    string                  `json:"id"`    // 工作流类型ID, 唯一标识, 用于标识工作流类型
	Name  string                  `json:"name"`  // 工作流类型名称
	Nodes []*NodeDefinitionConfig `json:"nodes"` // 构建工作流任务
}

// NodeDefinitionConfig 节点定义配置
type NodeDefinitionConfig struct {
	ID            string   `json:"id"`               // 节点ID, 唯一标识, 用于标识节点
	Name          string   `json:"name"`             // 节点名称
	NextNodes     []string `json:"next_nodes"`       // 后置节点ID列表
	FailMaxCount  *int64   `json:"fail_max_count"`   // 失败次数达到 fail_max_count次后，<0的忽略
	MaxWaitTimeTs *int64   `json:"max_wait_time_ts"` // 最大等待时间，单位秒，<=0 忽略
}

// RegisterWorkflowTaskParams 注册工作流任务节点参数
type RegisterWorkflowTaskParams struct {
	WorkflowType string
	TaskKey      string
	TaskWorker   WorkflowTaskNodeWorker
	IsPublic     bool // 是否公开，如果为true，则可以被其他工作流使用
}

/*
*
  - @description: 加载工作流配置
    只做存储使用，config转化会在CreateWorkflow中完成，延迟加载,主要是解决RegisterWorkflowTask的依赖
  - @param config *WorkflowConfig
  - @return error
*/
func LoadWorkflowConfig(config *WorkflowConfig) error {
	if config == nil {
		return errors.New("config is nil")
	}
	if _, ok := workflowDefinitions.Load(config.ID); ok {
		return errors.New(fmt.Sprintf("config already registered, id: %s", config.ID))
	}

	//todo 简单检查节点pre和next是否有问题
	workflowConfigs.Store(config.ID, config)
	return nil
}

/*
*
  - @description: 注册工作流任务节点
  - @param ctx context.Context
  - @param workflowType string
  - @param taskKey string
  - @param taskWorker WorkflowTaskWorker
  - @return error
    *
*/
func RegisterWorkflowTask(workflowType string, taskKey string, taskWorker WorkflowTaskNodeWorker) error {
	if taskWorker == nil {
		return errors.New("taskWorker is nil")
	}
	if _, ok := workflowTaskWorkers.Load(workflowType + "_" + taskKey); ok {
		return errors.New(fmt.Sprintf("taskWorker already registered, workflowType: %s, taskKey: %s", workflowType, taskKey))
	}
	workflowTaskWorkers.Store(workflowType+"_"+taskKey, taskWorker)
	return nil
}

func GetAndLoadWorkflowDefinition(workflowType string) (*WorkflowDefinition, error) {
	if i, ok := workflowDefinitions.Load(workflowType); ok {
		ret, ok := i.(*WorkflowDefinition)
		if !ok {
			return nil, errors.WithMessagef(ErrWorkflowDefinitionNotFound, "workflow definition not found, workflowType: %s, type error,please check code", workflowType)
		}
		return ret, nil
	}
	loadWorkflowLock.Lock()
	defer loadWorkflowLock.Unlock()
	if i, ok := workflowDefinitions.Load(workflowType); ok {
		ret, ok := i.(*WorkflowDefinition)
		if !ok {
			return nil, errors.WithMessagef(ErrWorkflowDefinitionNotFound, "workflow definition not found, workflowType: %s, type error,please check code", workflowType)
		}
		return ret, nil
	}
	// 加载配置处理
	workflowDefinitionCofigInterface, ok := workflowConfigs.Load(workflowType)
	if !ok {
		return nil, errors.WithMessagef(ErrWorkflowConfigNotFound, "workflow config %s not found", workflowType)
	}
	workflowDefinitionCofig, ok := workflowDefinitionCofigInterface.(*WorkflowConfig)
	if !ok {
		return nil, errors.WithMessagef(ErrWorkflowConfigNotFound, "workflow config %s not found, type error,please check code", workflowType)
	}

	rootNode := NewRootTaskNodeDefinition()
	endNode := NewEndTaskNodeDefinition()
	nodesMaps := make(map[string]*NodeDefinitionConfig)
	// 加上根节点和结束节点 所以+2
	nodeCount := int64(len(workflowDefinitionCofig.Nodes)) + 2
	nodeDefinitionConfigMap := make(map[string]*WorkflowTaskNodeDefinition)
	for _, node := range workflowDefinitionCofig.Nodes {
		nodesMaps[node.ID] = node
		workerFlowNodes := &WorkflowTaskNodeDefinition{
			TaskType:      node.ID,
			TaskName:      node.Name,
			FailMaxCount:  0,
			MaxWaitTimeTs: 0,
			PreNodes:      make([]*WorkflowTaskNodeDefinition, 0),
			NextNodes:     make([]*WorkflowTaskNodeDefinition, 0),
			TaskWorker:    defaultEmptyTaskWorker,
		}
		if node.FailMaxCount != nil {
			workerFlowNodes.FailMaxCount = *node.FailMaxCount
		}
		if node.MaxWaitTimeTs != nil {
			workerFlowNodes.MaxWaitTimeTs = *node.MaxWaitTimeTs
		}

		workerFlowNodes.TaskWorker, ok = getWorkflowTaskWorker(workflowType, node.ID)
		if !ok {
			return nil, errors.WithMessagef(ErrWorkflowTaskWorkerNotFound, "workflow task worker not found, workflowType: %s, taskKey: %s", workflowType, node.ID)
		}
		nodeDefinitionConfigMap[node.ID] = workerFlowNodes
	}

	// build 内部树结构, 挂载到endNode下面
	for _, node := range nodesMaps {
		// preNodes := make([]*WorkflowTaskNodeDefinition, 0)
		nextNodes := make([]*WorkflowTaskNodeDefinition, 0)
		for _, nextNode := range node.NextNodes {
			if _, ok := nodeDefinitionConfigMap[nextNode]; !ok {
				return nil, errors.WithMessagef(ErrWorkflowTaskWorkerNotFound, "workflow task worker not found, workflowType: %s, taskKey: %s", workflowType, nextNode)
			}
			nextNodes = append(nextNodes, nodeDefinitionConfigMap[nextNode])
		}
		BuildAddWorkfNode(endNode, nodeDefinitionConfigMap[node.ID], nextNodes)
	}
	// 挂载到rootNode下面
	for _, node := range nodeDefinitionConfigMap {
		// 如果前置节点为空,则直接挂载到rootNode下面
		if len(node.PreNodes) == 0 {
			rootNode.NextNodes = append(rootNode.NextNodes, node)
			node.PreNodes = append(node.PreNodes, rootNode)
		}
	}

	// todo 检查工作流图是否存在环, 所有路径都可到达
	err := checkNodeDefinitionIsOk(rootNode)
	if err != nil {
		return nil, errors.WithMessagef(err, "checkNodeDefinitionIsOk failed, workflowType: %s", workflowType)
	}

	workflowDefinition := &WorkflowDefinition{
		ID:         workflowType,
		Name:       workflowDefinitionCofig.Name,
		RootNode:   rootNode,
		NodesCount: nodeCount,
	}
	// 节点展开数组，方便后面节点处理
	nodes := make([]*WorkflowTaskNodeDefinition, 0)
	// 先加上根节点
	nodes = append(nodes, rootNode)
	for _, node := range workflowDefinitionCofig.Nodes {
		if _, ok := nodeDefinitionConfigMap[node.ID]; !ok {
			return nil, errors.WithMessagef(ErrWorkflowTaskWorkerNotFound, "workflow task worker not found, workflowType: %s, taskKey: %s", workflowType, node.ID)
		}
		nodes = append(nodes, nodeDefinitionConfigMap[node.ID])
	}
	// 加上结束节点
	nodes = append(nodes, endNode)
	workflowDefinition.Nodes = nodes
	workflowDefinitions.Store(workflowType, workflowDefinition)
	return workflowDefinition, nil
}

func BuildAddWorkfNode(endNode *WorkflowTaskNodeDefinition, addNode *WorkflowTaskNodeDefinition, nextNodes []*WorkflowTaskNodeDefinition) error {
	if addNode == nil {
		return errors.New("addNode is nil")
	}
	if addNode.TaskType == endTaskNode {
		// 如果当前节点是结束节点,则直接返回
		return nil
	}
	if len(nextNodes) == 0 {
		// 如果后置节点为空,是结束节点,next节点给空
		addNode.NextNodes = append(addNode.NextNodes, endNode)
		isNeedAddPreNode := true
		for _, nextNodePreNode := range endNode.PreNodes {
			if nextNodePreNode.TaskType == addNode.TaskType {
				isNeedAddPreNode = false
				break
			}
		}
		if isNeedAddPreNode {
			endNode.PreNodes = append(endNode.PreNodes, addNode)
		}
	} else {
		// 如果后置节点不为空，则将当前节点添加到后置节点的前置节点列表
		for _, nextNode := range nextNodes {
			isNeedAddPreNode := true
			for _, nextNodePreNode := range nextNode.PreNodes {
				if nextNodePreNode.TaskType == addNode.TaskType {
					isNeedAddPreNode = false
					break
				}
			}
			if isNeedAddPreNode {
				nextNode.PreNodes = append(nextNode.PreNodes, addNode)
			}
			addNode.NextNodes = append(addNode.NextNodes, nextNode)

		}
	}
	return nil
}

func (s *WorkflowServiceImpl) CreateWorkflow(ctx context.Context, req *CreateWorkflowReq) (*WorkflowInstance, error) {
	if err := validatorUtil.Struct(req); err != nil {
		return nil, errors.Wrapf(ErrWorkflowParamInvalid, "CreateWorkflow failed, req: %v,err: %v", req, err)
	}
	workflowDefinition, err := GetAndLoadWorkflowDefinition(req.WorkflowType)
	if err != nil && !req.IsRun {
		// 这里不用返回错误，创建和执行工作流的可能不在同一个容器里面，可能创建的容器中没有定义工作流
		// 需要记录日志
		slog.ErrorContext(ctx, "GetAndLoadWorkflowDefinition failed, workflowType: %s, err: %v", req.WorkflowType, err)
	}
	if err != nil && req.IsRun {
		// 需要立刻执行，说明创建和执行工作流在同一个容器里面，找不到工作流定义需要返回错误
		return nil, errors.WithMessagef(err, "GetAndLoadWorkflowDefinition failed, workflowType: %s", req.WorkflowType)
	}
	jsonContext := NewJSONContextFromMap(req.Context)

	workflowInstance, err := s.repo.CreateWorkflowInstance(ctx, &WorkflowInstancePo{
		WorkflowType:    req.WorkflowType,
		BusinessID:      req.BusinessID,
		WorkflowContext: jsonContext.ToBytesWithoutError(),
		Status:          WorkflowInstanceStatusInit,
		TaskId:          req.TaskId,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	})
	if err != nil {
		return nil, errors.WithMessagef(err, "CreateWorkflowInstance failed, workflowType: %s", req.WorkflowType)
	}

	if req.IsRun {
		// 如果需要立即执行，则需要执行工作流
		err = s.RunWorkflow(ctx, workflowInstance.ID)
		if err != nil {
			return nil, errors.WithMessagef(err, "RunWorkflow failed, workflowInstanceID: %d", workflowInstance.ID)
		}
	}
	return &WorkflowInstance{
		ID:              workflowInstance.ID,
		WorkflowType:    workflowInstance.WorkflowType,
		BusinessID:      workflowInstance.BusinessID,
		Status:          workflowInstance.Status,
		WorkflowContext: NewByte2StrctPbValue(workflowInstance.WorkflowContext),
		CreatedAt:       workflowInstance.CreatedAt,
		UpdatedAt:       workflowInstance.UpdatedAt,
		Definitions:     workflowDefinition,
		TaskId:          workflowInstance.TaskId,
	}, nil
}

func (s *WorkflowServiceImpl) CountWorkflowInstance(ctx context.Context, params *QueryWorkflowInstanceParams) (int64, error) {
	if err := validatorUtil.Struct(params); err != nil {
		return 0, errors.Wrapf(ErrWorkflowParamInvalid, "CountWorkflowInstance failed, params: %v,err: %v", params, err)
	}
	count, err := s.repo.CountWorkflowInstance(ctx, params)
	if err != nil {
		return 0, errors.WithMessagef(err, "QueryWorkflowInstance failed, params: %v", params)
	}
	return count, nil
}

func (s *WorkflowServiceImpl) QueryWorkflowInstanceDetail(ctx context.Context, params *QueryWorkflowInstanceParams) ([]*WorkflowInstanceDetailEntity, error) {
	if err := validatorUtil.Struct(params); err != nil {
		return nil, errors.Wrapf(ErrWorkflowParamInvalid, "QueryWorkflowInstanceDetail failed, params: %v,err: %v", params, err)
	}
	workflowInstances, err := s.repo.QueryWorkflowInstance(ctx, params)
	if err != nil {
		return nil, errors.WithMessagef(err, "QueryWorkflowInstance failed, params: %v", params)
	}

	workflowInstanceDetailEntities := make([]*WorkflowInstanceDetailEntity, 0)
	for _, workflowInstance := range workflowInstances {
		workflowInstanceDetailEntity, err := s.assemblyWorkflowInstanceDetailEntity(ctx, workflowInstance)
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("assemblyWorkflowInstanceDetailEntity failed, workflowInstanceID: %d, err: %v", workflowInstance.ID, err))
			continue
		}
		workflowInstanceDetailEntities = append(workflowInstanceDetailEntities, workflowInstanceDetailEntity)
	}
	return workflowInstanceDetailEntities, nil
}

func (s *WorkflowServiceImpl) QueryWorkflowInstancePo(ctx context.Context, params *QueryWorkflowInstanceParams) ([]*WorkflowInstancePo, error) {
	if err := validatorUtil.Struct(params); err != nil {
		return nil, errors.Wrapf(ErrWorkflowParamInvalid, "QueryWorkflowInstancePo failed, params: %v,err: %v", params, err)
	}
	workflowInstance, err := s.repo.QueryWorkflowInstance(ctx, params)
	if err != nil {
		return nil, errors.WithMessagef(err, "QueryWorkflowInstancePo failed, params: %v", params)
	}
	return workflowInstance, nil
}

func (s *WorkflowServiceImpl) assemblyWorkflowInstanceDetailEntity(ctx context.Context, workflowInstance *WorkflowInstancePo) (*WorkflowInstanceDetailEntity, error) {
	ret := &WorkflowInstanceDetailEntity{
		ID:              workflowInstance.ID,
		WorkflowType:    workflowInstance.WorkflowType,
		BusinessID:      workflowInstance.BusinessID,
		Status:          workflowInstance.Status,
		WorkflowContext: NewByte2StrctPbValue(workflowInstance.WorkflowContext),
		TaskInstances:   make([]*TaskInstanceEntity, 0),
		CreatedAt:       workflowInstance.CreatedAt,
		UpdatedAt:       workflowInstance.UpdatedAt,
	}
	workflowDefinition, err := GetAndLoadWorkflowDefinition(workflowInstance.WorkflowType)
	if err != nil {
		return nil, errors.WithMessagef(err, "GetAndLoadWorkflowDefinition failed, workflowType: %s", workflowInstance.WorkflowType)
	}
	// 查询任务实例
	taskInstances, err := s.getAllTaskInstancePo(ctx, workflowInstance.ID)
	if err != nil {
		return nil, errors.WithMessagef(err, "getAllTaskInstancePo failed, workflowInstanceID: %d", workflowInstance.ID)
	}
	taskMap := make(map[string]*WorkflowTaskInstancePo, 0)
	for _, taskInstance := range taskInstances {
		taskMap[taskInstance.TaskType] = taskInstance
	}
	for _, node := range workflowDefinition.Nodes {
		taskInstance, ok := taskMap[node.TaskType]
		preNodesKeys := make([]string, 0)
		nextNodesKeys := make([]string, 0)
		for _, preNode := range node.PreNodes {
			preNodesKeys = append(preNodesKeys, preNode.TaskType)
		}
		for _, nextNode := range node.NextNodes {
			nextNodesKeys = append(nextNodesKeys, nextNode.TaskType)
		}
		if ok {
			// 任务实例存在,使用任务实例的详情
			ret.TaskInstances = append(ret.TaskInstances, &TaskInstanceEntity{
				ID:                 taskInstance.ID,
				WorkflowInstanceID: workflowInstance.ID,
				TaskType:           taskInstance.TaskType,
				Status:             taskInstance.Status,
				TaskName:           node.TaskName,
				NodeContext:        NewByte2StrctPbValue(taskInstance.NodeContext),
				CreatedAt:          taskInstance.CreatedAt,
				UpdatedAt:          taskInstance.UpdatedAt,
				PreNodesKeys:       preNodesKeys,
				NextNodesKeys:      nextNodesKeys,
			})
		} else {
			// 任务实例不存在,使用节点定义的详情
			ret.TaskInstances = append(ret.TaskInstances, &TaskInstanceEntity{
				WorkflowInstanceID: workflowInstance.ID,
				TaskType:           node.TaskType,
				TaskName:           node.TaskName,
				Status:             WorkflowTaskNodeStatusStatusUnCreated,
				PreNodesKeys:       preNodesKeys,
				NextNodesKeys:      nextNodesKeys,
			})
		}
	}
	return ret, nil
}

func (s *WorkflowServiceImpl) getAllTaskInstancePo(ctx context.Context, workflowInstanceID int64) ([]*WorkflowTaskInstancePo, error) {
	fetchCount := 100
	page := 1
	retTaskInstances := make([]*WorkflowTaskInstancePo, 0)
	for {
		taskInstances, err := s.repo.QueryWorkflowTaskInstance(ctx, &QueryWorkflowTaskInstanceParams{
			WorkflowInstanceID: &workflowInstanceID,
			Page: &Pager{
				Page: int64(page),
				Size: int64(fetchCount),
			},
		})
		if err != nil {
			return nil, errors.WithMessagef(err, "QueryWorkflowTaskInstance failed, workflowInstanceID: %d", workflowInstanceID)
		}
		if len(taskInstances) == 0 {
			break
		}
		retTaskInstances = append(retTaskInstances, taskInstances...)
		if len(taskInstances) < fetchCount {
			break
		}
		page++
	}
	return retTaskInstances, nil
}

type AddNodeExternalEventParams struct {
	WorkflowInstanceID int64  `json:"workflow_instance_id" validate:"gt=0"`
	TaskType           string `json:"task_type" validate:"required"`
	// TaskInstanceID     int64              `json:"task_instance_id" validate:"required"`
	NodeEvent *NodeExternalEvent `json:"node_event" validate:"required"`
}
type NodeExternalEvent struct {
	// 事件时间,单位秒,会用来做版本控制，最新的版本会覆盖旧的版本
	EventTs      int64  `json:"event_ts"`
	EventContent string `json:"event_content"`
}
type RestartWorkflowNodeParams struct {
	WorkflowInstanceID      int64  `json:"workflow_instance_id" validate:"gt=0"`
	TaskType                string `json:"task_type" validate:"required"`
	IsForcedRestartWorkflow bool   `json:"is_forced_restart_workflow"` // 是否强制重启工作流,如果是true即使工作流实例已经结束,也会重启工作流
	// IsAsynchronous          bool   `json:"is_asynchronous" `           // 是否异步重启,如果为true，则不等待任务执行完成,直接返回
}

type RestartWorkflowParams struct {
	WorkflowInstanceID int64 `json:"workflow_instance_id" validate:"gt=0"`
	// Context            map[string]any // 上下文,如果有值，则覆盖掉原来的上下文
	IsRun bool // 是否立即执行,如果为true，则立即执行
}

func (s *WorkflowServiceImpl) RestartWorkflowNode(ctx context.Context, restartParams *RestartWorkflowNodeParams) error {
	if err := validatorUtil.Struct(restartParams); err != nil {
		return errors.Wrapf(ErrWorkflowParamInvalid, "RestartWorkflowNode failed, restartParams: %v,err: %v", restartParams, err)
	}
	err := s.executeLock.NonBlockingSynchronized(ctx,
		workflowOpLockKey(restartParams.WorkflowInstanceID),
		10*time.Minute,
		func(ctx context.Context) error {
			// 检查工作流节点是否存在
			workflowInstance, err := s.repo.QueryWorkflowInstance(ctx, &QueryWorkflowInstanceParams{
				WorkflowInstanceID: &restartParams.WorkflowInstanceID,
				Page: &Pager{
					Page: 1,
					Size: 1,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			if len(workflowInstance) == 0 {
				return errors.Errorf("WorkflowInstance not found, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			definition, err := GetAndLoadWorkflowDefinition(workflowInstance[0].WorkflowType)
			if err != nil {
				return errors.WithMessagef(err, "GetAndLoadWorkflowDefinition failed, workflowType: %s", workflowInstance[0].WorkflowType)
			}
			hasNode := false
			currentNode := definition.RootNode
			// 检查节点是否存在
			for _, node := range definition.Nodes {
				if node.TaskType == restartParams.TaskType {
					hasNode = true
					currentNode = node
					break
				}
			}
			if !hasNode {
				return errors.Errorf("WorkflowTaskNode not found, workflowInstanceID: %d, taskType: %s", restartParams.WorkflowInstanceID, restartParams.TaskType)
			}
			allChildrenTaskType := getAllChildrenTaskType(currentNode)
			resetTaskTypeMap := make(map[string]struct{})
			for _, taskType := range allChildrenTaskType {
				resetTaskTypeMap[taskType] = struct{}{}
			}
			resetTaskTypeMap[restartParams.TaskType] = struct{}{}

			// 事务处理 工作流状态和工作流节点实例状态
			err = s.repo.Transaction(ctx, func(ctx context.Context) error {
				if IsOverWorkflowInstanceStatus(workflowInstance[0].Status) {
					// 工作流实例已经结束,
					if !restartParams.IsForcedRestartWorkflow {
						return errors.Errorf("WorkflowInstance is over, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
					}
					// 强制重启工作流
					workflowInstance[0].Status = WorkflowInstanceStatusRunning
					err = s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
						Where: &UpdateWorkflowInstanceWhere{
							IDIn: []int64{restartParams.WorkflowInstanceID},
						},
						Fields: &UpdateWorkflowInstanceField{
							Status: &workflowInstance[0].Status,
						},
						LimitMax: 1,
					})
					if err != nil {
						return errors.WithMessagef(err, "UpdateWorkflowInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
					}
				}
				// 查询任务实例
				taskInstances, err := s.getAllTaskInstancePo(ctx, restartParams.WorkflowInstanceID)
				if err != nil {
					return errors.WithMessagef(err, "QueryWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", restartParams.WorkflowInstanceID, restartParams.TaskType)
				}
				if len(taskInstances) == 0 {
					// 任务实例不存在,直接返回nil,说明这个节点没有执行过
					return nil
				}
				resetTaskIds := make([]int64, 0)
				for _, taskInstance := range taskInstances {
					if _, ok := resetTaskTypeMap[taskInstance.TaskType]; ok {
						resetTaskIds = append(resetTaskIds, taskInstance.ID)
					}
				}
				if len(resetTaskIds) > 0 {
					err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
						Where: &UpdateWorkflowTaskInstanceWhere{
							IDIn: resetTaskIds,
						},
						Fields: &UpdateWorkflowTaskInstanceField{
							Status: String(WorkflowTaskNodeStatusRestarting),
						},
						LimitMax: len(resetTaskIds),
					})
					if err != nil {
						return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", restartParams.WorkflowInstanceID, restartParams.TaskType)
					}
				}
				return nil
			})
			if err != nil {
				return errors.WithMessagef(err, "RestartWorkflowNode failed, restartParams: %v", restartParams)
			}
			return nil
		})
	if err != nil {
		return errors.WithMessagef(err, "RestartWorkflowNode failed, restartParams: %v", restartParams)
	}

	return nil
}
func (s *WorkflowServiceImpl) RestartWorkflowInstance(ctx context.Context, restartParams *RestartWorkflowParams) error {
	if err := validatorUtil.Struct(restartParams); err != nil {
		return errors.Wrapf(ErrWorkflowParamInvalid, "RestartWorkflowInstance failed, restartParams: %v,err: %v", restartParams, err)
	}
	err := s.executeLock.NonBlockingSynchronized(ctx,
		workflowOpLockKey(restartParams.WorkflowInstanceID),
		10*time.Minute,
		func(ctx context.Context) error {
			workflowInstance, err := s.repo.QueryWorkflowInstance(ctx, &QueryWorkflowInstanceParams{
				WorkflowInstanceID: &restartParams.WorkflowInstanceID,
				Page: &Pager{
					Page: 1,
					Size: 1,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			if len(workflowInstance) == 0 {
				return errors.Errorf("WorkflowInstance not found, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			if !IsOverWorkflowInstanceStatus(workflowInstance[0].Status) {
				// 工作流实例状态未结束,直接返回nil
				return nil
			}
			workflowInstance[0].Status = WorkflowInstanceStatusRunning

			err = s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
				Where: &UpdateWorkflowInstanceWhere{
					IDIn: []int64{restartParams.WorkflowInstanceID},
				},
				Fields: &UpdateWorkflowInstanceField{
					Status: &workflowInstance[0].Status,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "UpdateWorkflowInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			taskInstances, err := s.getAllTaskInstancePo(ctx, restartParams.WorkflowInstanceID)
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowTaskInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
			}
			resetTaskIds := make([]int64, 0)
			for _, taskInstance := range taskInstances {
				if taskInstance.Status == WorkflowTaskNodeStatusFailed || taskInstance.Status == WorkflowTaskNodeStatusCancelled {
					resetTaskIds = append(resetTaskIds, taskInstance.ID)
				}
			}
			// 重新启动任务实例
			if len(resetTaskIds) > 0 {
				err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
					Where: &UpdateWorkflowTaskInstanceWhere{
						IDIn: resetTaskIds,
					},
					Fields: &UpdateWorkflowTaskInstanceField{
						Status: String(WorkflowTaskNodeStatusRestarting),
					},
					LimitMax: len(resetTaskIds),
				})
				if err != nil {
					return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
				}
			}
			if restartParams.IsRun {
				err = s.RunWorkflow(ctx, restartParams.WorkflowInstanceID)
				if err != nil {
					return errors.WithMessagef(err, "RunWorkflow failed, workflowInstanceID: %d", restartParams.WorkflowInstanceID)
				}
			}
			return nil
		})
	if err != nil {
		return errors.WithMessagef(err, "RestartWorkflowInstance failed, restartParams: %v", restartParams)
	}
	return nil
}

// 添加节点外部事件,会覆盖掉旧的事件
func (s *WorkflowServiceImpl) AddNodeExternalEvent(ctx context.Context, addParams *AddNodeExternalEventParams) error {
	if err := validatorUtil.Struct(addParams); err != nil {
		return errors.Wrapf(ErrWorkflowParamInvalid, "AddNodeExternalEvent failed, addParams: %v,err: %v", addParams, err)
	}
	err := s.executeLock.NonBlockingSynchronized(ctx,
		workflowOpLockKey(addParams.WorkflowInstanceID),
		10*time.Minute,
		func(ctx context.Context) error {
			// 查询任务实例
			taskInstances, err := s.repo.QueryWorkflowTaskInstance(ctx, &QueryWorkflowTaskInstanceParams{
				WorkflowInstanceID: &addParams.WorkflowInstanceID,
				TaskType:           &addParams.TaskType,
				OrderbyIDAsc:       Bool(false),
				Page: &Pager{
					Page: 1,
					Size: 2,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}
			if len(taskInstances) == 0 {
				// 任务实例不存在，可能没有准备好，需要等他init后
				return errors.Errorf("WorkflowTaskInstance not found, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}
			if len(taskInstances) >= 2 {
				// 任务实例大于2，可能有重复事件，需要开发人员去确认是否有问题的
				// 后续看迭代升级，目前设计上不存在同一个工作流里面有两个相同的任务
				return errors.Errorf("WorkflowTaskInstance is more than 2, please check, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}

			if IsOverWorkflowTaskNodeStatus(taskInstances[0].Status) {
				// 任务实例状态已经结束，不能再触发
				return errors.Errorf("WorkflowTaskInstance status is error, please check, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}
			newNodeContext := NewByte2StrctPbValue(taskInstances[0].NodeContext)

			// 获取现有的事件信息
			eventEvent := &NodeExternalEvent{}
			if _, ok := newNodeContext.Get(NodeContextKeyNodeEvent); ok {
				// 尝试解析现有的 node_event
				if nodeEventValue, ok := newNodeContext.Get(NodeContextKeyNodeEvent); ok {
					if nodeEventMap, ok := nodeEventValue.(map[string]any); ok {
						if ts, ok := nodeEventMap["event_ts"].(float64); ok {
							eventEvent.EventTs = int64(ts)
						}
						if content, ok := nodeEventMap["event_content"].(string); ok {
							eventEvent.EventContent = content
						}
					}
				}
			}

			// 检查时间戳
			if eventEvent.EventTs > addParams.NodeEvent.EventTs {
				return errors.Errorf("EventTs is less than the latest eventTs, please check, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}

			// 更新事件信息
			newNodeContext.Set([]string{NodeContextKeyNodeEvent, "event_content"}, addParams.NodeEvent.EventContent)
			newNodeContext.Set([]string{NodeContextKeyNodeEvent, "event_ts"}, addParams.NodeEvent.EventTs)

			err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
				Where: &UpdateWorkflowTaskInstanceWhere{
					IDIn: []int64{taskInstances[0].ID},
				},
				Fields: &UpdateWorkflowTaskInstanceField{
					NodeContext: newNodeContext,
				},
				LimitMax: 1,
			})
			if err != nil {
				return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", addParams.WorkflowInstanceID, addParams.TaskType)
			}
			return nil
		})
	if errors.Is(err, LockFailedError) {
		return errors.Errorf("LockFailedError,workflowInstanceID: %d, taskType: %s, err: %v", addParams.WorkflowInstanceID, addParams.TaskType, err)
	}
	return err
}

func workflowOpLockKey(workflowInstanceID int64) string {
	return fmt.Sprintf("workflow_instance_execute_%d", workflowInstanceID)
}

type WorkflowInstanceDetailEntity struct {
	ID              int64
	WorkflowType    string
	BusinessID      string
	Status          WorkflowInstanceStatus
	WorkflowContext *JSONContext
	CreatedAt       int64
	UpdatedAt       int64
	TaskInstances   []*TaskInstanceEntity
}

type TaskInstanceEntity struct {
	ID                 int64 //ID 可能为0,因为还没有创建
	WorkflowInstanceID int64
	TaskType           string
	TaskName           string
	Status             string
	NodeContext        *JSONContext
	CreatedAt          int64
	UpdatedAt          int64
	PreNodesKeys       []string
	NextNodesKeys      []string
}

type WorkflowInstance struct {
	ID              int64
	WorkflowType    string
	BusinessID      string
	Status          string
	WorkflowContext *JSONContext
	TaskId          int64
	CreatedAt       int64
	UpdatedAt       int64
	Definitions     *WorkflowDefinition
}

func (s *WorkflowServiceImpl) RunWorkflow(ctx context.Context, workflowID int64) error {
	if workflowID <= 0 {
		return errors.Wrapf(ErrWorkflowParamInvalid, "RunWorkflow failed, workflowID: %d", workflowID)
	}
	workflowInstances, err := s.repo.QueryWorkflowInstance(ctx, &QueryWorkflowInstanceParams{
		WorkflowInstanceID: &workflowID,
		Page: &Pager{
			Page: 1,
			Size: 1,
		},
	})
	if err != nil {
		return errors.WithMessagef(err, "QueryWorkflowInstance failed, workflowID: %d", workflowID)
	}
	if len(workflowInstances) == 0 {
		return errors.WithMessagef(ErrWorkflowInstanceNotFound, "WorkflowInstance not found, workflowID: %d", workflowID)
	}
	workflowInstance := &WorkflowInstance{
		ID:              workflowInstances[0].ID,
		WorkflowType:    workflowInstances[0].WorkflowType,
		BusinessID:      workflowInstances[0].BusinessID,
		Status:          workflowInstances[0].Status,
		WorkflowContext: NewByte2StrctPbValue(workflowInstances[0].WorkflowContext),
		CreatedAt:       workflowInstances[0].CreatedAt,
		UpdatedAt:       workflowInstances[0].UpdatedAt,
		Definitions:     nil,
	}
	workflowDefinition, err := GetAndLoadWorkflowDefinition(workflowInstance.WorkflowType)
	if err != nil {
		return errors.WithMessagef(err, "GetAndLoadWorkflowDefinition failed, workflowType: %s", workflowInstance.WorkflowType)
	}
	workflowInstance.Definitions = workflowDefinition

	err = s.executeLock.NonBlockingSynchronized(ctx,
		workflowOpLockKey(workflowInstance.ID),
		10*time.Minute,
		func(ctx context.Context) error {
			// 查询任务实例
			taskInstanceNodes, err := s.repo.QueryWorkflowTaskInstance(ctx, &QueryWorkflowTaskInstanceParams{
				WorkflowInstanceID: &workflowInstance.ID,
				Page: &Pager{
					Page: 1,
					Size: workflowDefinition.NodesCount + 1,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowTaskInstance failed, workflowInstanceID: %d", workflowInstance.ID)
			}
			if len(taskInstanceNodes) > int(workflowDefinition.NodesCount) {
				// 如果任务实例节点大于节点数量，目前的框架上是任务有问题的,需要开发人员去确认是否有问题的
				slog.ErrorContext(ctx, fmt.Sprintf("WorkflowTaskInstance node is more than nodes count,please check, workflowInstanceID: %d", workflowInstance.ID))
			}
			taskNodeMap := make(map[string]*WorkflowTaskNode, 0)
			for _, taskInstanceNode := range taskInstanceNodes {
				taskNodeMap[taskInstanceNode.TaskType] = &WorkflowTaskNode{
					ID:                 taskInstanceNode.ID,
					WorkflowInstanceID: taskInstanceNode.WorkflowInstanceID,
					TaskType:           taskInstanceNode.TaskType,
					Status:             taskInstanceNode.Status,
					NodeContext:        NewByte2StrctPbValue(taskInstanceNode.NodeContext),
					CreatedAt:          taskInstanceNode.CreatedAt,
					UpdatedAt:          taskInstanceNode.UpdatedAt,
					FailCount:          taskInstanceNode.FailCount,
				}
			}
			err = s.visitTaskNodeAndExecute(ctx, workflowInstance, workflowDefinition.RootNode, &taskNodeMap)
			if err != nil && errors.Is(err, ErrWorkflowTaskFailedWithFailed) {
				// 工作流取消
				batchCancelTaskIDs := make([]int64, 0)
				for _, taskNode := range taskNodeMap {
					if taskNode.Status == WorkflowInstanceStatusInit || taskNode.Status == WorkflowInstanceStatusRunning {
						// 工作流被取消，需要批量更新任务实例状态为取消
						batchCancelTaskIDs = append(batchCancelTaskIDs, taskNode.ID)
					}
				}
				if len(batchCancelTaskIDs) > 0 {
					err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
						Where: &UpdateWorkflowTaskInstanceWhere{
							IDIn: batchCancelTaskIDs,
						},
						Fields: &UpdateWorkflowTaskInstanceField{
							// 批量更新任务实例状态为取消
							Status: String(WorkflowTaskNodeStatusCancelled),
						},
						LimitMax: len(batchCancelTaskIDs),
					})
				}
			}

			return err

		})
	if err != nil {
		return errors.WithMessagef(err, "NonBlockingSynchronized failed, workflowInstanceID: %d", workflowInstance.ID)
	}
	return nil
}

func (s *WorkflowServiceImpl) CancelWorkflowInstance(ctx context.Context, workflowInstanceID int64) error {
	if workflowInstanceID <= 0 {
		return errors.Wrapf(ErrWorkflowParamInvalid, "CancelWorkflowInstance failed, workflowInstanceID: %d", workflowInstanceID)
	}
	return s.executeLock.NonBlockingSynchronized(ctx,
		workflowOpLockKey(workflowInstanceID),
		10*time.Minute,
		func(ctx context.Context) error {
			workflowInstance, err := s.repo.QueryWorkflowInstance(ctx, &QueryWorkflowInstanceParams{
				WorkflowInstanceID: &workflowInstanceID,
				Page: &Pager{
					Page: 1,
					Size: 1,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "QueryWorkflowInstance failed, workflowInstanceID: %d", workflowInstanceID)
			}
			if len(workflowInstance) == 0 {
				return errors.WithMessagef(ErrWorkflowInstanceNotFound, "WorkflowInstance not found, workflowInstanceID: %d", workflowInstanceID)
			}
			if IsOverWorkflowInstanceStatus(workflowInstance[0].Status) {
				return nil
			}
			err = s.repo.Transaction(ctx, func(ctx context.Context) error {
				err = s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
					Where: &UpdateWorkflowInstanceWhere{
						IDIn: []int64{workflowInstanceID},
						StatusIn: []string{
							workflowInstance[0].Status,
						},
					},
					Fields: &UpdateWorkflowInstanceField{
						Status: String(WorkflowInstanceStatusCancelled),
					},
					LimitMax: 1,
				})
				if err != nil {
					return errors.WithMessagef(err, "UpdateWorkflowInstance failed, workflowInstanceID: %d", workflowInstanceID)
				}
				taskInstances, err := s.getAllTaskInstancePo(ctx, workflowInstanceID)
				if err != nil {
					return errors.WithMessagef(err, "getAllTaskInstancePo failed, workflowInstanceID: %d", workflowInstanceID)
				}
				updateTaskIDs := make([]int64, 0)
				for _, taskInstance := range taskInstances {
					if IsOverWorkflowTaskNodeStatus(taskInstance.Status) {
						continue
					}
					updateTaskIDs = append(updateTaskIDs, taskInstance.ID)
				}
				if len(updateTaskIDs) > 0 {
					err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
						Where: &UpdateWorkflowTaskInstanceWhere{
							IDIn: updateTaskIDs,
						},
						Fields: &UpdateWorkflowTaskInstanceField{
							Status: String(WorkflowTaskNodeStatusCancelled),
						},
						LimitMax: len(updateTaskIDs),
					})
					if err != nil {
						return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d", workflowInstanceID)
					}
				}
				return nil
			})
			return nil
		})
}

// 访问节点并执行
// 原则：除触发工作流取消情况提前返回，其余情况都会访问所有的可达节点
// 本函数有递归操作，不要使用defer，不要使用defer，不要使用defer，重要的事情说三遍
func (s *WorkflowServiceImpl) visitTaskNodeAndExecute(ctx context.Context, workflowInstance *WorkflowInstance, rootNode *WorkflowTaskNodeDefinition, taskNodeMap *map[string]*WorkflowTaskNode) error {
	if rootNode == nil {
		// 不会出现这种情况
		return errors.New("rootNode is nil")
	}
	// 先访问当前节点
	taskNode, ok := (*taskNodeMap)[rootNode.TaskType]
	if !ok {
		// 节点不存在,需要节点需要初始化
		isInit := true
		preTasklist := make([]*WorkflowTaskNode, 0)
		for _, preNode := range rootNode.PreNodes {
			preNodeInstance, ok := (*taskNodeMap)[preNode.TaskType]
			if !ok {
				// 前置节点不存在，当前节点不用执行，直接跳过
				isInit = false
				break
			} else {
				// 前置节点存在,需要检查前置节点是否完成
				if preNodeInstance.Status != WorkflowTaskNodeStatusCompleted {
					isInit = false
					break
				}
				preTasklist = append(preTasklist, &WorkflowTaskNode{
					ID:                 preNodeInstance.ID,
					WorkflowInstanceID: preNodeInstance.WorkflowInstanceID,
					TaskType:           preNodeInstance.TaskType,
					Status:             preNodeInstance.Status,
					NodeContext:        preNodeInstance.NodeContext,
					UpdatedAt:          preNodeInstance.UpdatedAt,
					FailCount:          preNodeInstance.FailCount,
				})
			}
		}
		if !isInit {
			// 当前节点不满足初始化条件，直接跳过
			return nil
		}

		preNodeAllContext := make(map[string]interface{})
		for _, preTask := range preTasklist {
			preNodeMap := preTask.NodeContext.ToMap()
			// 删除pre_node_context和workflow_context,不用追溯到上层
			delete(preNodeMap, "pre_node_context") // 不追溯到上层
			delete(preNodeMap, "workflow_context") // 冗余字段
			delete(preNodeMap, "system")           // 上层的系统相关参数不用保存
			preNodeAllContext[preTask.TaskType] = preNodeMap
		}

		newNodeContext := NewJSONContextFromMap(map[string]any{
			"pre_node_context": preNodeAllContext,
			"workflow_context": workflowInstance.WorkflowContext.ToMap(),
		})
		if rootNode.TaskType == rootTaskNode {
			// 根节点初始化,需要额外将workflowInstance 状态转化为runing
			workflowInstance.Status = WorkflowInstanceStatusRunning
			err := s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
				Where: &UpdateWorkflowInstanceWhere{
					IDIn: []int64{workflowInstance.ID},
				},
				Fields: &UpdateWorkflowInstanceField{
					Status: &workflowInstance.Status,
				},
				LimitMax: 1,
			})
			if err != nil {
				return errors.WithMessagef(err, "UpdateWorkflowInstance failed, workflowInstanceID: %d", workflowInstance.ID)
			}
		}
		// 创建任务实例
		taskInstancePo, err := s.repo.CreateWorkflowTaskInstance(ctx, &WorkflowTaskInstancePo{
			WorkflowInstanceID: workflowInstance.ID,
			TaskType:           rootNode.TaskType,
			Status:             WorkflowInstanceStatusRunning, // 直接设置为running即可
			NodeContext:        newNodeContext.ToBytesWithoutError(),
			CreatedAt:          time.Now().Unix(),
			UpdatedAt:          time.Now().Unix(),
		})
		if err != nil {
			return errors.WithMessagef(err, "CreateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
		}

		taskInstanceNode := &WorkflowTaskNode{
			ID:                 taskInstancePo.ID,
			WorkflowInstanceID: taskInstancePo.WorkflowInstanceID,
			TaskType:           taskInstancePo.TaskType,
			Status:             taskInstancePo.Status,
			NodeContext:        NewByte2StrctPbValue(taskInstancePo.NodeContext),
			CreatedAt:          taskInstancePo.CreatedAt,
			UpdatedAt:          taskInstancePo.UpdatedAt,
			FailCount:          taskInstancePo.FailCount,
		}
		(*taskNodeMap)[rootNode.TaskType] = taskInstanceNode
		if err := s.taskRun(ctx, workflowInstance, rootNode, taskInstanceNode); err != nil {
			if errors.Is(err, ErrWorkflowTaskFailedWithFailed) {
				return errors.WithMessagef(err, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
			}
			if IsSeriousError(err) {
				slog.ErrorContext(ctx, fmt.Sprintf("[error]TaskRun failed, workflowInstanceID: %d, taskType: %s, err: %v", workflowInstance.ID, rootNode.TaskType, err))
			} else {
				slog.WarnContext(ctx, fmt.Sprintf("[warn]TaskRun failed, workflowInstanceID: %d, taskType: %s, err: %v", workflowInstance.ID, rootNode.TaskType, err))
			}
			// 其他的错误，需要记录日志,不需要返回错误，继续check后续节点
			return nil
		}
	} else {
		// 当前节点存在，需要检查当前节点是否完成
		if taskNode.Status == WorkflowTaskNodeStatusCompleted {
			// 当前节点已完成,不需要执行该节点,遍历他的子节点即可
			for _, nextNode := range rootNode.NextNodes {
				err := s.visitTaskNodeAndExecute(ctx, workflowInstance, nextNode, taskNodeMap)
				if err != nil {
					return errors.WithMessagef(err, "visitTaskNodeAndExecute failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
				}
			}
			return nil
		}
		// 当前节点未完成，需要执行该节点
		if taskNode.Status == WorkflowTaskNodeStatusCancelled || taskNode.Status == WorkflowTaskNodeStatusFailed {
			// 当前节点已取消或者失败，直接返回,不需要再处理
			if workflowInstance.Status == WorkflowInstanceStatusCancelled || workflowInstance.Status == WorkflowInstanceStatusFailed {
				// 工作流已经取消，直接返回
				return errors.WithMessagef(ErrWorkflowTaskFailedWithFailed, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
			}
			originalStatus := workflowInstance.Status
			if taskNode.Status == WorkflowTaskNodeStatusCancelled {
				// 工作流标记为取消
				workflowInstance.Status = WorkflowInstanceStatusCancelled
			} else {
				// 工作流标记为失败
				workflowInstance.Status = WorkflowInstanceStatusFailed
			}
			workflowInstance.UpdatedAt = time.Now().Unix()
			newErr := s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
				Where: &UpdateWorkflowInstanceWhere{
					IDIn: []int64{workflowInstance.ID},
				},
				Fields: &UpdateWorkflowInstanceField{
					Status: &workflowInstance.Status,
				},
				LimitMax: 1,
			})
			if newErr != nil {
				workflowInstance.Status = originalStatus
				slog.ErrorContext(ctx, fmt.Sprintf("UpdateWorkflowInstance failed,err: %v", newErr))
			}
			return errors.WithMessagef(ErrWorkflowTaskFailedWithFailed, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
		}
		// 重新启动状态，当作重新初始化处理
		if taskNode.Status == WorkflowTaskNodeStatusRestarting {

			// 任务实例初始化状态,需要重新启动
			isReady := true
			preTasklist := make([]*WorkflowTaskNode, 0)
			for _, preNode := range rootNode.PreNodes {
				preNodeInstance, ok := (*taskNodeMap)[preNode.TaskType]
				if !ok {
					// 前置节点不存在，当前节点不用执行，直接跳过
					isReady = false
					break
				} else {
					// 前置节点存在,需要检查前置节点是否完成
					if preNodeInstance.Status != WorkflowTaskNodeStatusCompleted {
						isReady = false
						break
					}
					preTasklist = append(preTasklist, preNodeInstance)
				}
			}
			if !isReady {
				// 任务实例未准备好，直接返回
				return nil
			}
			// 重新初始化节点上下文
			taskNode.Status = WorkflowTaskNodeStatusRunning
			preNodeAllContext := make(map[string]interface{})
			for _, preTask := range preTasklist {
				preNodeMap := preTask.NodeContext.ToMap()
				delete(preNodeMap, "pre_node_context") // 不追溯到上层
				delete(preNodeMap, "workflow_context") // 冗余字段
				delete(preNodeMap, "system")           // 上层的系统相关参数不用保存
				preNodeAllContext[preTask.TaskType] = preNodeMap
			}
			newNodeContext := NewJSONContextFromMap(map[string]any{
				"pre_node_context": preNodeAllContext,
				"workflow_context": workflowInstance.WorkflowContext.ToMap(),
			})
			err := s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
				Where: &UpdateWorkflowTaskInstanceWhere{
					IDIn: []int64{taskNode.ID},
				},
				Fields: &UpdateWorkflowTaskInstanceField{
					NodeContext: newNodeContext,
					Status:      &taskNode.Status,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "CreateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
			}

		}
		err := s.taskRun(ctx, workflowInstance, rootNode, taskNode)
		if err != nil {
			if errors.Is(err, ErrWorkflowTaskFailedWithFailed) {
				return errors.WithMessagef(err, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
			}
			if IsSeriousError(err) {
				slog.ErrorContext(ctx, fmt.Sprintf("[error]TaskRun failed, workflowInstanceID: %d, taskType: %s, err: %v", workflowInstance.ID, rootNode.TaskType, err))
			} else {
				slog.WarnContext(ctx, fmt.Sprintf("[warn]TaskRun failed, workflowInstanceID: %d, taskType: %s, err: %v", workflowInstance.ID, rootNode.TaskType, err))
			}
			// 其他的错误，需要记录日志,不需要返回错误，继续check后续节点
			return nil
		}
	}
	// 当前节点完成，不需要再处理，处理后续节点
	for _, nextNode := range rootNode.NextNodes {
		err := s.visitTaskNodeAndExecute(ctx, workflowInstance, nextNode, taskNodeMap)
		if err != nil {
			return errors.WithMessagef(err, "visitTaskNodeAndExecute failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, rootNode.TaskType)
		}
	}

	return nil

}

func (s *WorkflowServiceImpl) taskRun(ctx context.Context, workflowInstance *WorkflowInstance, taskNode *WorkflowTaskNodeDefinition, taskInstance *WorkflowTaskNode) (err error) {
	if taskNode == nil {
		return errors.New("taskNode is nil")
	}
	if taskInstance == nil {
		return errors.New("taskInstance is nil")
	}
	// 针对不同的错误给不同的处理逻辑
	defer func() {
		// panic 捕捉一下，返回给上方
		if r := recover(); r != nil {
			stack := debug.Stack()
			slog.ErrorContext(ctx, "taskRun panic: %v, task InstanceID: %d, taskType: %s, stack: %w", r, taskInstance.ID, taskNode.TaskType, string(stack))
			err = errors.New(fmt.Sprintf("taskRun panic: %v, task InstanceID: %d, taskType: %s", r, taskInstance.ID, taskNode.TaskType))
		}
		// 添加错误信息到节点上下文
		if err != nil {
			// 添加错误信息到节点上下文
			s.addTaskNodeContextSystemError(err, taskInstance.NodeContext)
			// 检查是否有超出时间了
			if taskNode.MaxWaitTimeTs > 0 && time.Now().Unix()-taskInstance.CreatedAt > taskNode.MaxWaitTimeTs {
				// 超出时间，取消工作流
				err = errors.WithMessagef(ErrWorkflowTaskFailedWithFailed, "TaskRun failed with cancel, timeout, taskInstanceID: %d, workflowInstanceID: %d, taskType: %s,err: %v", taskInstance.ID, workflowInstance.ID, taskNode.TaskType, err)
				reasonvalue, ok := taskInstance.NodeContext.GetString(NodeContextKeyReason)
				if !ok || reasonvalue == "" {
					taskInstance.NodeContext.Set([]string{NodeContextKeyReason}, "任务节点执行超时")
				}
			}

			// 任务实例没有准备好，直接返回，这个error不算error
			if errors.Is(err, ErrorWorkflowTaskInstanceNotReady) {
				// 任务实例没有准备好，直接返回
				// 正常的业务行为,不需要处理
				err = nil
				// 额外保存一下nodecontext, not reday可能会保存一些数据处理
				err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
					Where: &UpdateWorkflowTaskInstanceWhere{
						IDIn: []int64{taskInstance.ID},
					},
					Fields: &UpdateWorkflowTaskInstanceField{
						NodeContext: taskInstance.NodeContext,
					},
					LimitMax: 1,
				})
				return
			}
			// 任务实例失败，但是可以继续执行，当作完成处理
			if errors.Is(err, ErrorWorkflowTaskFailedWithContinue) {
				taskInstance.Status = WorkflowTaskNodeStatusCompleted
				taskInstance.UpdatedAt = time.Now().Unix()
				taskInstance.FailCount++
				// 	这个有报错，就不处理了
				err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
					Where: &UpdateWorkflowTaskInstanceWhere{
						IDIn: []int64{taskInstance.ID},
					},
					Fields: &UpdateWorkflowTaskInstanceField{
						Status:      &taskInstance.Status,
						NodeContext: taskInstance.NodeContext,
						FailCount:   &taskInstance.FailCount,
					},
					LimitMax: 1,
				})
				return
			}

			// 检查是否需要增加失败次数
			if taskNode.FailMaxCount > 0 && taskInstance.FailCount+1 >= taskNode.FailMaxCount {
				// 检查失败次数是否达到极限
				// 换error了重新赋值一下error
				err = errors.WithMessagef(ErrWorkflowTaskFailedWithFailed, "TaskRun failed with cancel, fail max count, taskInstanceID: %d, workflowInstanceID: %d, taskType: %s,err: %v", taskInstance.ID, workflowInstance.ID, taskNode.TaskType, err)
				s.addTaskNodeContextSystemError(err, taskInstance.NodeContext)
			}

			// 任务实例失败，整个工作流状态变成取消错误
			if errors.Is(err, ErrWorkflowTaskFailedWithFailed) {
				// task 任务转化为cancle
				originalStatus := taskInstance.Status
				taskInstance.Status = WorkflowTaskNodeStatusFailed
				taskInstance.FailCount++
				taskInstance.UpdatedAt = time.Now().Unix()
				newErr := s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
					Where: &UpdateWorkflowTaskInstanceWhere{
						IDIn: []int64{taskInstance.ID},
					},
					Fields: &UpdateWorkflowTaskInstanceField{
						Status:      &taskInstance.Status,
						NodeContext: taskInstance.NodeContext,
						FailCount:   &taskInstance.FailCount,
					},
					LimitMax: 1,
				})
				if newErr != nil {
					taskInstance.Status = originalStatus
					err = errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed,err: %v", newErr)
					return
				}

				if workflowInstance.Status == WorkflowInstanceStatusCancelled || workflowInstance.Status == WorkflowInstanceStatusFailed {
					// 工作流已经取消，直接返回nil
					err = errors.WithMessagef(err, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
					return
				}

				originalStatus = workflowInstance.Status
				workflowInstance.Status = WorkflowInstanceStatusFailed
				workflowInstance.UpdatedAt = time.Now().Unix()
				newErr = s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
					Where: &UpdateWorkflowInstanceWhere{
						IDIn: []int64{workflowInstance.ID},
					},
					Fields: &UpdateWorkflowInstanceField{
						Status: &workflowInstance.Status,
					},
					LimitMax: 1,
				})
				if newErr != nil {
					// 简单回滚一下状态
					workflowInstance.Status = originalStatus
					slog.ErrorContext(ctx, fmt.Sprintf("UpdateWorkflowInstance failed,err: %v", newErr))
				}
				err = errors.WithMessagef(err, "TaskRun failed with cancel, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
				return
			}

			// 其他的err 情况，需要记录失败次数
			taskInstance.FailCount++
			taskInstance.UpdatedAt = time.Now().Unix()
			newErr := s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
				Where: &UpdateWorkflowTaskInstanceWhere{
					IDIn: []int64{taskInstance.ID},
				},
				Fields: &UpdateWorkflowTaskInstanceField{
					FailCount:   &taskInstance.FailCount,
					NodeContext: taskInstance.NodeContext,
				},
				LimitMax: 1,
			})
			if newErr != nil {
				err = errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed,err: %v", newErr)
			}
			return

		}
	}()
	if taskInstance.Status == WorkflowTaskNodeStatusRunning {
		err := taskNode.TaskWorker.Run(ctx, taskInstance.NodeContext)
		if err != nil {
			return errors.WithMessagef(err, "Run failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
		}
		// 更新任务实例状态到pending
		taskInstance.Status = WorkflowTaskNodeStatusPending
		taskInstance.UpdatedAt = time.Now().Unix()
		err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
			Where: &UpdateWorkflowTaskInstanceWhere{
				IDIn: []int64{taskInstance.ID},
			},
			Fields: &UpdateWorkflowTaskInstanceField{
				Status:      &taskInstance.Status,
				NodeContext: taskInstance.NodeContext,
			},
			LimitMax: 1,
		})
		if err != nil {
			return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
		}
	}
	if taskInstance.Status == WorkflowTaskNodeStatusPending {
		err := taskNode.TaskWorker.AsynchronousWaitCheck(ctx, taskInstance.NodeContext)
		if err != nil {
			return errors.WithMessagef(err, "AsynchronousWaitCheck failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
		}
		taskInstance.Status = WorkflowTaskNodeStatusFinishing
		taskInstance.UpdatedAt = time.Now().Unix()

		err = s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
			Where: &UpdateWorkflowTaskInstanceWhere{
				IDIn: []int64{taskInstance.ID},
			},
			Fields: &UpdateWorkflowTaskInstanceField{
				Status:      &taskInstance.Status,
				NodeContext: taskInstance.NodeContext,
			},
			LimitMax: 1,
		})
		if err != nil {
			return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
		}
	}
	if taskInstance.Status == WorkflowTaskNodeStatusFinishing {
		taskInstance.Status = WorkflowTaskNodeStatusCompleted
		taskInstance.UpdatedAt = time.Now().Unix()
		err := s.repo.UpdateWorkflowTaskInstance(ctx, &UpdateWorkflowTaskInstanceParams{
			Where: &UpdateWorkflowTaskInstanceWhere{
				IDIn: []int64{taskInstance.ID},
			},
			Fields: &UpdateWorkflowTaskInstanceField{
				Status: &taskInstance.Status,
			},
			LimitMax: 1,
		})
		if err != nil {
			return errors.WithMessagef(err, "UpdateWorkflowTaskInstance failed, workflowInstanceID: %d, taskType: %s", workflowInstance.ID, taskNode.TaskType)
		}
		if taskNode.TaskType == endTaskNode {
			// 根节点完成,需要额外将workflowInstance 状态转化为completed
			workflowInstance.Status = WorkflowInstanceStatusCompleted
			workflowInstance.UpdatedAt = time.Now().Unix()
			err := s.repo.UpdateWorkflowInstance(ctx, &UpdateWorkflowInstanceParams{
				Where: &UpdateWorkflowInstanceWhere{
					IDIn: []int64{workflowInstance.ID},
				},
				Fields: &UpdateWorkflowInstanceField{
					Status: &workflowInstance.Status,
				},
			})
			if err != nil {
				return errors.WithMessagef(err, "UpdateWorkflowInstance failed, workflowInstanceID: %d", workflowInstance.ID)
			}
		}

	}
	return nil
}

func (s *WorkflowServiceImpl) addTaskNodeContextSystemError(err error, nodeContext *JSONContext) {
	if nodeContext == nil {
		return
	}
	if err == nil {
		return
	}
	// 设置系统错误信息
	nodeContext.Set([]string{"system", "last_error"}, err.Error())
	nodeContext.Set([]string{"system", "last_error_time"}, time.Now().Format(time.RFC3339))
}

func checkNodeDefinitionIsOk(rootNode *WorkflowTaskNodeDefinition) error {
	if rootNode == nil {
		return errors.New("rootNode is nil")
	}
	visitMap := make(map[string]bool)
	return visitNodeDefinition(rootNode, visitMap)
}
func visitNodeDefinition(rootNode *WorkflowTaskNodeDefinition, visitMap map[string]bool) error {
	if rootNode == nil {
		return errors.New("rootNode is nil")
	}
	if rootNode.TaskType == endTaskNode {
		// 达到终点，返回,说明没有环，可以正常执行正常的路径
		return nil
	}
	// 不是终点，则需要检查是否已经访问过
	if visitMap[rootNode.TaskType] {
		// 已经访问过，说明有环，不能正常执行，返回错误
		return errors.New("rootNode，taskType: " + rootNode.TaskType + " is already visited, there is a cycle in the workflow")
	}

	// 没有访问过，则需要标记为已访问
	visitMap[rootNode.TaskType] = true

	// 递归访问下一个节点
	for _, nextNode := range rootNode.NextNodes {
		err := visitNodeDefinition(nextNode, visitMap)
		if err != nil {
			return errors.WithMessagef(err, "visitNodeDefinition failed, rootNode: %s, nextNode: %s", rootNode.TaskType, nextNode.TaskType)
		}
	}
	visitMap[rootNode.TaskType] = false
	return nil
}

func NewByte2StrctPbValue(b []byte) *JSONContext {
	return NewJSONContext(b)
}

func PreloadingWorkflowDefinition() error {
	allWorkflowTypes := make([]string, 0)
	errorlist := make([]error, 0)
	var err error
	workflowConfigs.Range(func(key, value interface{}) bool {
		workflowType, ok := key.(string)
		if !ok {
			err = errors.New("workflowType is not string")
			return true
		}
		allWorkflowTypes = append(allWorkflowTypes, workflowType)
		return true
	})
	if err != nil {
		return errors.WithMessagef(err, "PreloadingWorkflowDefinition failed")
	}
	for _, workflowType := range allWorkflowTypes {
		_, err := GetAndLoadWorkflowDefinition(workflowType)
		if err != nil {
			errorlist = append(errorlist, err)
		}
	}
	if len(errorlist) > 0 {
		return goerrors.Join(errorlist...)
	}
	return nil
}

func StructUnmarshal(ctx *JSONContext, v any) error {
	return ctx.Unmarshal(v)
}
