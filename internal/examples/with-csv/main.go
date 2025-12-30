package main

// csv作为工作流基础的数据源

import (
	"context"
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/blingmoon/simple-workflow/internal/commonregister"
	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/pkg/errors"
)

var _ workflow.WorkflowRepo = (*CsvRepo)(nil)

type CsvRepo struct {
	workflowInstanceFile string
	taskInstanceFile     string
	mu                   sync.RWMutex
}

// NewCsvRepo 创建 CSV 存储实现
// workflowInstanceFile: WorkflowInstancePo 对应的 CSV 文件路径，如 "workflow_instance.csv"
// taskInstanceFile: WorkflowTaskInstancePo 对应的 CSV 文件路径，如 "task_instance.csv"
func NewCsvRepo(workflowInstanceFile, taskInstanceFile string) *CsvRepo {
	repo := &CsvRepo{
		workflowInstanceFile: workflowInstanceFile,
		taskInstanceFile:     taskInstanceFile,
	}
	// 初始化 CSV 文件，如果不存在则创建并写入表头
	repo.initCSVFiles()
	return repo
}

// initCSVFiles 初始化 CSV 文件，如果不存在则创建并写入表头
func (c *CsvRepo) initCSVFiles() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 初始化 workflow_instance.csv
	if _, err := os.Stat(c.workflowInstanceFile); os.IsNotExist(err) {
		c.writeWorkflowInstanceHeader()
	}

	// 初始化 task_instance.csv
	if _, err := os.Stat(c.taskInstanceFile); os.IsNotExist(err) {
		c.writeTaskInstanceHeader()
	}
}

// writeWorkflowInstanceHeader 写入 workflow_instance.csv 表头
func (c *CsvRepo) writeWorkflowInstanceHeader() {
	file, err := os.Create(c.workflowInstanceFile)
	if err != nil {
		return
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"id", "workflow_type", "business_id", "status", "workflow_context", "task_id", "created_at", "updated_at"})
}

// writeTaskInstanceHeader 写入 task_instance.csv 表头
func (c *CsvRepo) writeTaskInstanceHeader() {
	file, err := os.Create(c.taskInstanceFile)
	if err != nil {
		return
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"id", "workflow_instance_id", "task_type", "status", "fail_count", "node_context", "created_at", "updated_at"})
}

// readWorkflowInstances 读取所有 workflow instance
func (c *CsvRepo) readWorkflowInstances() ([]*workflow.WorkflowInstancePo, error) {
	file, err := os.Open(c.workflowInstanceFile)
	if err != nil {
		return nil, errors.WithMessage(err, "open workflow instance file failed")
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.WithMessage(err, "read workflow instance CSV failed")
	}

	if len(records) < 2 {
		return []*workflow.WorkflowInstancePo{}, nil
	}

	instances := make([]*workflow.WorkflowInstancePo, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 8 {
			continue
		}

		id, _ := strconv.ParseInt(record[0], 10, 64)
		taskID, _ := strconv.ParseInt(record[5], 10, 64)
		createdAt, _ := strconv.ParseInt(record[6], 10, 64)
		updatedAt, _ := strconv.ParseInt(record[7], 10, 64)

		instances = append(instances, &workflow.WorkflowInstancePo{
			ID:              id,
			WorkflowType:    record[1],
			BusinessID:      record[2],
			Status:          workflow.WorkflowInstanceStatus(record[3]),
			WorkflowContext: []byte(record[4]),
			TaskId:          taskID,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		})
	}
	return instances, nil
}

// writeWorkflowInstances 写入所有 workflow instance
func (c *CsvRepo) writeWorkflowInstances(instances []*workflow.WorkflowInstancePo) error {
	file, err := os.Create(c.workflowInstanceFile)
	if err != nil {
		return errors.WithMessage(err, "create workflow instance file failed")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"id", "workflow_type", "business_id", "status", "workflow_context", "task_id", "created_at", "updated_at"})

	// 写入数据
	for _, inst := range instances {
		writer.Write([]string{
			strconv.FormatInt(inst.ID, 10),
			inst.WorkflowType,
			inst.BusinessID,
			string(inst.Status),
			string(inst.WorkflowContext),
			strconv.FormatInt(inst.TaskId, 10),
			strconv.FormatInt(inst.CreatedAt, 10),
			strconv.FormatInt(inst.UpdatedAt, 10),
		})
	}
	return nil
}

// readTaskInstances 读取所有 task instance
func (c *CsvRepo) readTaskInstances() ([]*workflow.WorkflowTaskInstancePo, error) {
	file, err := os.Open(c.taskInstanceFile)
	if err != nil {
		return nil, errors.WithMessage(err, "open task instance file failed")
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.WithMessage(err, "read task instance CSV failed")
	}

	if len(records) < 2 {
		return []*workflow.WorkflowTaskInstancePo{}, nil
	}

	tasks := make([]*workflow.WorkflowTaskInstancePo, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 8 {
			continue
		}

		id, _ := strconv.ParseInt(record[0], 10, 64)
		workflowInstanceID, _ := strconv.ParseInt(record[1], 10, 64)
		failCount, _ := strconv.ParseInt(record[4], 10, 64)
		createdAt, _ := strconv.ParseInt(record[6], 10, 64)
		updatedAt, _ := strconv.ParseInt(record[7], 10, 64)

		tasks = append(tasks, &workflow.WorkflowTaskInstancePo{
			ID:                 id,
			WorkflowInstanceID: workflowInstanceID,
			TaskType:           record[2],
			Status:             workflow.WorkflowTaskNodeStatus(record[3]),
			FailCount:          failCount,
			NodeContext:        []byte(record[5]),
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
		})
	}
	return tasks, nil
}

// writeTaskInstances 写入所有 task instance
func (c *CsvRepo) writeTaskInstances(tasks []*workflow.WorkflowTaskInstancePo) error {
	file, err := os.Create(c.taskInstanceFile)
	if err != nil {
		return errors.WithMessage(err, "create task instance file failed")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"id", "workflow_instance_id", "task_type", "status", "fail_count", "node_context", "created_at", "updated_at"})

	// 写入数据
	for _, task := range tasks {
		writer.Write([]string{
			strconv.FormatInt(task.ID, 10),
			strconv.FormatInt(task.WorkflowInstanceID, 10),
			task.TaskType,
			string(task.Status),
			strconv.FormatInt(task.FailCount, 10),
			string(task.NodeContext),
			strconv.FormatInt(task.CreatedAt, 10),
			strconv.FormatInt(task.UpdatedAt, 10),
		})
	}
	return nil
}

// getNextWorkflowInstanceID 获取下一个 workflow instance ID
func (c *CsvRepo) getNextWorkflowInstanceID() (int64, error) {
	instances, err := c.readWorkflowInstances()
	if err != nil {
		return 0, err
	}
	maxID := int64(0)
	for _, inst := range instances {
		if inst.ID > maxID {
			maxID = inst.ID
		}
	}
	return maxID + 1, nil
}

// getNextTaskInstanceID 获取下一个 task instance ID
func (c *CsvRepo) getNextTaskInstanceID() (int64, error) {
	tasks, err := c.readTaskInstances()
	if err != nil {
		return 0, err
	}
	maxID := int64(0)
	for _, task := range tasks {
		if task.ID > maxID {
			maxID = task.ID
		}
	}
	return maxID + 1, nil
}

// filterWorkflowInstances 过滤 workflow instances
func (c *CsvRepo) filterWorkflowInstances(instances []*workflow.WorkflowInstancePo, param *workflow.QueryWorkflowInstanceParams) []*workflow.WorkflowInstancePo {
	result := make([]*workflow.WorkflowInstancePo, 0)
	for _, inst := range instances {
		if param.WorkflowInstanceID != nil && inst.ID != *param.WorkflowInstanceID {
			continue
		}
		if len(param.WorkflowTypeIn) > 0 {
			found := false
			for _, wt := range param.WorkflowTypeIn {
				if inst.WorkflowType == wt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if param.BusinessID != nil && inst.BusinessID != *param.BusinessID {
			continue
		}
		if len(param.StatusIn) > 0 {
			found := false
			for _, status := range param.StatusIn {
				if string(inst.Status) == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if param.IDGreaterThan != nil && inst.ID <= *param.IDGreaterThan {
			continue
		}
		if param.TaskID != nil && inst.TaskId != *param.TaskID {
			continue
		}
		result = append(result, inst)
	}
	return result
}

// filterTaskInstances 过滤 task instances
func (c *CsvRepo) filterTaskInstances(tasks []*workflow.WorkflowTaskInstancePo, param *workflow.QueryWorkflowTaskInstanceParams) []*workflow.WorkflowTaskInstancePo {
	result := make([]*workflow.WorkflowTaskInstancePo, 0)
	for _, task := range tasks {
		if param.WorkflowTaskInstanceID != nil && task.ID != *param.WorkflowTaskInstanceID {
			continue
		}
		if param.WorkflowInstanceID != nil && task.WorkflowInstanceID != *param.WorkflowInstanceID {
			continue
		}
		if param.TaskType != nil && task.TaskType != *param.TaskType {
			continue
		}
		if len(param.StatusIn) > 0 {
			found := false
			for _, status := range param.StatusIn {
				if string(task.Status) == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if param.IDGreaterThan != nil && task.ID <= *param.IDGreaterThan {
			continue
		}
		result = append(result, task)
	}
	return result
}

// CountWorkflowInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) CountWorkflowInstance(ctx context.Context, param *workflow.QueryWorkflowInstanceParams) (int64, error) {
	if param == nil {
		return 0, errors.New("nil QueryWorkflowInstanceParams")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	instances, err := c.readWorkflowInstances()
	if err != nil {
		return 0, err
	}

	filtered := c.filterWorkflowInstances(instances, param)
	return int64(len(filtered)), nil
}

// CreateWorkflowInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) CreateWorkflowInstance(ctx context.Context, workflowInstance *workflow.WorkflowInstancePo) (*workflow.WorkflowInstancePo, error) {
	if workflowInstance == nil {
		return nil, errors.New("nil WorkflowInstancePo")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	instances, err := c.readWorkflowInstances()
	if err != nil {
		return nil, err
	}

	// 生成 ID
	if workflowInstance.ID == 0 {
		workflowInstance.ID, err = c.getNextWorkflowInstanceID()
		if err != nil {
			return nil, err
		}
	}

	// 设置时间戳
	now := time.Now().Unix()
	if workflowInstance.CreatedAt == 0 {
		workflowInstance.CreatedAt = now
	}
	workflowInstance.UpdatedAt = now

	// 添加到列表
	instances = append(instances, workflowInstance)

	// 写回文件
	if err := c.writeWorkflowInstances(instances); err != nil {
		return nil, err
	}

	return workflowInstance, nil
}

// CreateWorkflowTaskInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) CreateWorkflowTaskInstance(ctx context.Context, workflowTaskInstance *workflow.WorkflowTaskInstancePo) (*workflow.WorkflowTaskInstancePo, error) {
	if workflowTaskInstance == nil {
		return nil, errors.New("nil WorkflowTaskInstancePo")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	tasks, err := c.readTaskInstances()
	if err != nil {
		return nil, err
	}

	// 生成 ID
	if workflowTaskInstance.ID == 0 {
		workflowTaskInstance.ID, err = c.getNextTaskInstanceID()
		if err != nil {
			return nil, err
		}
	}

	// 设置时间戳
	now := time.Now().Unix()
	if workflowTaskInstance.CreatedAt == 0 {
		workflowTaskInstance.CreatedAt = now
	}
	workflowTaskInstance.UpdatedAt = now

	// 添加到列表
	tasks = append(tasks, workflowTaskInstance)

	// 写回文件
	if err := c.writeTaskInstances(tasks); err != nil {
		return nil, err
	}

	return workflowTaskInstance, nil
}

// QueryWorkflowInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) QueryWorkflowInstance(ctx context.Context, param *workflow.QueryWorkflowInstanceParams) ([]*workflow.WorkflowInstancePo, error) {
	if param == nil {
		return nil, errors.New("nil QueryWorkflowInstanceParams")
	}
	if param.Page == nil {
		return nil, errors.New("page is nil")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	instances, err := c.readWorkflowInstances()
	if err != nil {
		return nil, err
	}

	// 过滤
	filtered := c.filterWorkflowInstances(instances, param)

	// 排序
	if param.OrderbyIDAsc != nil {
		if *param.OrderbyIDAsc {
			// 升序
			for i := 0; i < len(filtered)-1; i++ {
				for j := i + 1; j < len(filtered); j++ {
					if filtered[i].ID > filtered[j].ID {
						filtered[i], filtered[j] = filtered[j], filtered[i]
					}
				}
			}
		} else {
			// 降序
			for i := 0; i < len(filtered)-1; i++ {
				for j := i + 1; j < len(filtered); j++ {
					if filtered[i].ID < filtered[j].ID {
						filtered[i], filtered[j] = filtered[j], filtered[i]
					}
				}
			}
		}
	}

	// 分页
	if param.Page.IsNoLimit == nil || !*param.Page.IsNoLimit {
		page := param.Page.Page
		if page == 0 {
			page = 1
		}
		size := param.Page.Size
		if size == 0 {
			size = 10
		}
		start := (page - 1) * size
		end := start + size
		if start >= int64(len(filtered)) {
			return []*workflow.WorkflowInstancePo{}, nil
		}
		if end > int64(len(filtered)) {
			end = int64(len(filtered))
		}
		filtered = filtered[start:end]
	}

	return filtered, nil
}

// QueryWorkflowTaskInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) QueryWorkflowTaskInstance(ctx context.Context, param *workflow.QueryWorkflowTaskInstanceParams) ([]*workflow.WorkflowTaskInstancePo, error) {
	if param == nil {
		return nil, errors.New("nil QueryWorkflowTaskInstanceParams")
	}
	if param.Page == nil {
		return nil, errors.New("page is nil")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	tasks, err := c.readTaskInstances()
	if err != nil {
		return nil, err
	}

	// 过滤
	filtered := c.filterTaskInstances(tasks, param)

	// 排序
	if param.OrderbyIDAsc != nil {
		if *param.OrderbyIDAsc {
			// 升序
			for i := 0; i < len(filtered)-1; i++ {
				for j := i + 1; j < len(filtered); j++ {
					if filtered[i].ID > filtered[j].ID {
						filtered[i], filtered[j] = filtered[j], filtered[i]
					}
				}
			}
		} else {
			// 降序
			for i := 0; i < len(filtered)-1; i++ {
				for j := i + 1; j < len(filtered); j++ {
					if filtered[i].ID < filtered[j].ID {
						filtered[i], filtered[j] = filtered[j], filtered[i]
					}
				}
			}
		}
	}

	// 分页
	page := param.Page.Page
	if page == 0 {
		page = 1
	}
	size := param.Page.Size
	if size == 0 {
		size = 10
	}
	start := (page - 1) * size
	end := start + size
	if start >= int64(len(filtered)) {
		return []*workflow.WorkflowTaskInstancePo{}, nil
	}
	if end > int64(len(filtered)) {
		end = int64(len(filtered))
	}
	filtered = filtered[start:end]

	return filtered, nil
}

// UpdateWorkflowInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) UpdateWorkflowInstance(ctx context.Context, param *workflow.UpdateWorkflowInstanceParams) error {
	if param == nil {
		return errors.New("nil UpdateWorkflowInstanceParams")
	}
	if param.Where == nil {
		return errors.New("where is nil")
	}
	if param.Fields == nil {
		return errors.New("fields is nil")
	}
	if len(param.Where.IDIn) == 0 && len(param.Where.StatusIn) == 0 {
		return errors.New("update workflow instance need where condition")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	instances, err := c.readWorkflowInstances()
	if err != nil {
		return err
	}

	updated := 0
	now := time.Now().Unix()
	for _, inst := range instances {
		shouldUpdate := false

		// 检查 where 条件
		if len(param.Where.IDIn) > 0 {
			for _, id := range param.Where.IDIn {
				if inst.ID == id {
					shouldUpdate = true
					break
				}
			}
		}
		if !shouldUpdate && len(param.Where.StatusIn) > 0 {
			for _, status := range param.Where.StatusIn {
				if string(inst.Status) == status {
					shouldUpdate = true
					break
				}
			}
		}

		if shouldUpdate {
			if param.Fields.Status != nil {
				inst.Status = workflow.WorkflowInstanceStatus(*param.Fields.Status)
			}
			if param.Fields.WorkflowContext != nil {
				inst.WorkflowContext, _ = param.Fields.WorkflowContext.ToBytes()
			}
			inst.UpdatedAt = now
			updated++
			if updated >= param.LimitMax {
				break
			}
		}
	}

	return c.writeWorkflowInstances(instances)
}

// UpdateWorkflowTaskInstance implements workflow.WorkflowRepo.
func (c *CsvRepo) UpdateWorkflowTaskInstance(ctx context.Context, param *workflow.UpdateWorkflowTaskInstanceParams) error {
	if param == nil {
		return errors.New("nil UpdateWorkflowTaskInstanceParams")
	}
	if param.Where == nil {
		return errors.New("where is nil")
	}
	if param.Fields == nil {
		return errors.New("fields is nil")
	}
	if len(param.Where.IDIn) == 0 {
		return errors.New("update workflow task instance need where condition")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	tasks, err := c.readTaskInstances()
	if err != nil {
		return err
	}

	updated := 0
	now := time.Now().Unix()
	for _, task := range tasks {
		shouldUpdate := false

		// 检查 where 条件
		for _, id := range param.Where.IDIn {
			if task.ID == id {
				shouldUpdate = true
				break
			}
		}

		if shouldUpdate {
			if param.Fields.Status != nil {
				task.Status = workflow.WorkflowTaskNodeStatus(*param.Fields.Status)
			}
			if param.Fields.NodeContext != nil {
				task.NodeContext, _ = param.Fields.NodeContext.ToBytes()
			}
			if param.Fields.FailCount != nil {
				task.FailCount = *param.Fields.FailCount
			}
			task.UpdatedAt = now
			updated++
			if updated >= param.LimitMax {
				break
			}
		}
	}

	return c.writeTaskInstances(tasks)
}

// Transaction implements workflow.WorkflowRepo.
// CSV 文件不支持真正的事务，这里简单执行函数
func (c *CsvRepo) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// CSV 文件不支持事务，直接执行函数
	// 注意：这不是真正的事务，如果中间出错，已写入的数据不会回滚
	return fn(ctx)
}

func main() {
	repo := NewCsvRepo("workflow_instance.csv", "task_instance.csv")

	service := workflow.NewWorkflowService(repo, workflow.NewLocalWorkflowLock())

	// 注册审批工作流任务
	if err := commonregister.RegisterApprovalInstanceTask(service); err != nil {
		panic(err)
	}

	// 4. 创建工作流实例
	_, err := service.CreateWorkflow(context.Background(), &workflow.CreateWorkflowReq{
		WorkflowType: "approval_workflow",
		BusinessID:   "ORDER-2024-001",
		Context: map[string]any{
			"order_id":     "ORDER-2024-001",
			"amount":       1000.00,
			"created_time": time.Now().Unix(),
		},
		IsRun: true,
	})
	if err != nil {
		panic(err)
	}

}
