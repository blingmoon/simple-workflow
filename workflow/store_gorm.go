package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type WorkflowInstancePo struct {
	ID              int64                  `gorm:"column:id;primaryKey;autoIncrement" json:"id"	`
	WorkflowType    string                 `gorm:"column:workflow_type" json:"workflow_type"`
	BusinessID      string                 `gorm:"column:business_id" json:"business_id"`
	Status          WorkflowInstanceStatus `gorm:"column:status" json:"status"`
	WorkflowContext []byte                 `gorm:"column:workflow_context" json:"workflow_context"` // 工作流上下文
	TaskId          int64                  `gorm:"column:task_id" json:"task_id"`
	CreatedAt       int64                  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       int64                  `gorm:"column:updated_at" json:"updated_at"`
}

func (WorkflowInstancePo) TableName() string {
	return "workflow_instance"
}

type WorkflowTaskInstancePo struct {
	ID                 int64                  `gorm:"column:id;primaryKey;autoIncrement"`
	WorkflowInstanceID int64                  `gorm:"column:workflow_instance_id"`
	TaskType           string                 `gorm:"column:task_type"`
	Status             WorkflowTaskNodeStatus `gorm:"column:status"`
	FailCount          int64                  `gorm:"column:fail_count"`
	NodeContext        []byte                 `gorm:"column:node_context"` // 节点上下文, input,output结合在一起
	CreatedAt          int64                  `gorm:"column:created_at"`
	UpdatedAt          int64                  `gorm:"column:updated_at"`
}

func (WorkflowTaskInstancePo) TableName() string {
	return "task_instance"
}

type QueryWorkflowInstanceParams struct {
	WorkflowInstanceID *int64   `json:"workflow_instance_id"`
	WorkflowTypeIn     []string `json:"workflow_type_in"`
	BusinessID         *string  `json:"business_id"`
	StatusIn           []string `json:"status_in"`
	IDGreaterThan      *int64   `json:"id_greater_than"`
	TaskID             *int64   `json:"task_id"`
	OrderbyIDAsc       *bool    `json:"orderby_id_asc"`
	Page               *Pager   `json:"page"`
}

type Pager struct {
	IsNoLimit *bool `json:"is_no_limit"`
	Page      int64 `json:"page"`
	Size      int64 `json:"size"`
}

type QueryWorkflowTaskInstanceParams struct {
	WorkflowTaskInstanceID *int64   `json:"workflow_task_instance_id"`
	WorkflowInstanceID     *int64   `json:"workflow_instance_id"`
	TaskType               *string  `json:"task_type"`
	StatusIn               []string `json:"status_in"`
	IDGreaterThan          *int64   `json:"id_greater_than"`
	OrderbyIDAsc           *bool    `json:"orderby_id_asc"`
	Page                   *Pager   `json:"page"`
}

type UpdateWorkflowInstanceParams struct {
	Where    *UpdateWorkflowInstanceWhere `json:"where" validate:"required"`
	Fields   *UpdateWorkflowInstanceField `json:"field" validate:"required"`
	LimitMax int                          `json:"limit_max" validate:"required"`
}

type UpdateWorkflowInstanceWhere struct {
	IDIn     []int64  `json:"id_in"`
	StatusIn []string `json:"status_in"`
}

type UpdateWorkflowInstanceField struct {
	Status          *string      `json:"status"`
	WorkflowContext *JSONContext `json:"workflow_context"`
}

type UpdateWorkflowTaskInstanceParams struct {
	Where    *UpdateWorkflowTaskInstanceWhere `json:"where" validate:"required"`
	Fields   *UpdateWorkflowTaskInstanceField `json:"field" validate:"required"`
	LimitMax int                              `json:"limit_max" validate:"required"`
}

type UpdateWorkflowTaskInstanceWhere struct {
	IDIn []int64 `json:"id_in"`
}

type UpdateWorkflowTaskInstanceField struct {
	Status      *string      `json:"status"`
	NodeContext *JSONContext `json:"node_context"`
	FailCount   *int64       `json:"fail_count"`
}

type workflowRepo struct {
	db *gorm.DB
}

func NewWorkflowRepo(db *gorm.DB) WorkflowRepo {
	return &workflowRepo{
		db: db,
	}
}

func (r *workflowRepo) CreateWorkflowInstance(ctx context.Context, workflowInstance *WorkflowInstancePo) (*WorkflowInstancePo, error) {
	if workflowInstance == nil {
		return nil, fmt.Errorf("nil WorkflowInstancePo")
	}
	workflowInstance.CreatedAt = time.Now().Unix()
	workflowInstance.UpdatedAt = time.Now().Unix()
	if err := r.GetDBWithContext(ctx).Create(workflowInstance).Error; err != nil {
		return nil, errors.WithMessage(err, "CreateWorkflowInstance failed")
	}
	return workflowInstance, nil
}

func (r *workflowRepo) CreateWorkflowTaskInstance(ctx context.Context, workflowTaskInstance *WorkflowTaskInstancePo) (*WorkflowTaskInstancePo, error) {
	if workflowTaskInstance == nil {
		return nil, errors.New("nil WorkflowTaskInstancePo")
	}
	workflowTaskInstance.CreatedAt = time.Now().Unix()
	workflowTaskInstance.UpdatedAt = time.Now().Unix()
	if err := r.GetDBWithContext(ctx).Create(workflowTaskInstance).Error; err != nil {
		return nil, errors.WithMessage(err, "CreateWorkflowTaskInstance failed")
	}
	return workflowTaskInstance, nil
}

func buildQueryWorkflowInstanceParams(db *gorm.DB, isCount bool, param *QueryWorkflowInstanceParams) (*gorm.DB, error) {
	if param == nil {
		return nil, errors.New("nil QueryWorkflowInstanceParams")
	}
	if param.WorkflowInstanceID != nil {
		db = db.Where("id = ?", param.WorkflowInstanceID)
	}
	if len(param.WorkflowTypeIn) != 0 {
		db = db.Where("workflow_type IN ?", param.WorkflowTypeIn)
	}
	if param.BusinessID != nil {
		db = db.Where("business_id = ?", param.BusinessID)
	}
	if len(param.StatusIn) != 0 {
		db = db.Where("status IN ?", param.StatusIn)
	}
	if param.IDGreaterThan != nil {
		db = db.Where("id > ?", param.IDGreaterThan)
	}
	if param.TaskID != nil {
		db = db.Where("task_id = ?", param.TaskID)
	}
	if param.OrderbyIDAsc != nil && !isCount {
		// 排序处理
		if *param.OrderbyIDAsc {
			db = db.Order("id asc")
		} else {
			db = db.Order("id desc")
		}
	}
	if !isCount {
		if param.Page == nil {
			return nil, errors.New("page is nil")
		}
		if param.Page.IsNoLimit != nil && *param.Page.IsNoLimit {
			// 不分页显示指定了true
			return db, nil
		}
		if param.Page.Page == 0 {
			param.Page.Page = 1
		}
		if param.Page.Size == 0 {
			param.Page.Size = 10
		}
		db = db.Offset(int(param.Page.Page-1) * int(param.Page.Size)).Limit(int(param.Page.Size))
	}
	return db, nil
}

func (r *workflowRepo) QueryWorkflowInstance(ctx context.Context, param *QueryWorkflowInstanceParams) ([]*WorkflowInstancePo, error) {
	if param == nil {
		return nil, fmt.Errorf("nil QueryWorkflowInstanceParams")
	}
	db := r.GetDBWithContext(ctx).Model(&WorkflowInstancePo{})
	db, err := buildQueryWorkflowInstanceParams(db, false, param)
	if err != nil {
		return nil, errors.WithMessage(err, "buildQueryWorkflowInstanceParams failed")
	}
	pos := make([]*WorkflowInstancePo, 0)
	if err := db.Find(&pos).Error; err != nil {
		return nil, errors.WithMessage(err, "QueryWorkflowInstance failed")
	}
	return pos, nil
}
func (r *workflowRepo) CountWorkflowInstance(ctx context.Context, param *QueryWorkflowInstanceParams) (int64, error) {
	if param == nil {
		return 0, fmt.Errorf("nil QueryWorkflowInstanceParams")
	}
	db := r.GetDBWithContext(ctx).Model(&WorkflowInstancePo{})
	db, err := buildQueryWorkflowInstanceParams(db, true, param)
	if err != nil {
		return 0, errors.WithMessage(err, "buildQueryWorkflowInstanceParams failed")
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, errors.WithMessage(err, "CountWorkflowInstance failed")
	}
	return count, nil
}

func buildQueryWorkflowTaskInstanceParams(db *gorm.DB, isCount bool, param *QueryWorkflowTaskInstanceParams) (*gorm.DB, error) {
	if param == nil {
		return nil, errors.New("nil QueryWorkflowTaskInstanceParams")
	}
	if param.WorkflowTaskInstanceID != nil {
		db = db.Where("id = ?", param.WorkflowTaskInstanceID)
	}
	if param.WorkflowInstanceID != nil {
		db = db.Where("workflow_instance_id = ?", param.WorkflowInstanceID)
	}
	if param.TaskType != nil {
		db = db.Where("task_type = ?", param.TaskType)
	}
	if len(param.StatusIn) != 0 {
		db = db.Where("status IN ?", param.StatusIn)
	}
	if param.IDGreaterThan != nil {
		db = db.Where("id > ?", param.IDGreaterThan)
	}
	if param.OrderbyIDAsc != nil {
		if *param.OrderbyIDAsc {
			db = db.Order("id asc")
		} else {
			db = db.Order("id desc")
		}
	}
	if !isCount {
		if param.Page == nil {
			return nil, errors.New("page is nil")
		}
		if param.Page.Page == 0 {
			param.Page.Page = 1
		}
		if param.Page.Size == 0 {
			param.Page.Size = 10
		}
		db = db.Offset(int(param.Page.Page-1) * int(param.Page.Size)).Limit(int(param.Page.Size))
	}
	return db, nil
}
func (r *workflowRepo) QueryWorkflowTaskInstance(ctx context.Context, param *QueryWorkflowTaskInstanceParams) ([]*WorkflowTaskInstancePo, error) {
	if param == nil {
		return nil, fmt.Errorf("nil QueryWorkflowTaskInstanceParams")
	}
	db := r.GetDBWithContext(ctx).Model(&WorkflowTaskInstancePo{})
	db, err := buildQueryWorkflowTaskInstanceParams(db, false, param)
	if err != nil {
		return nil, errors.WithMessage(err, "buildQueryWorkflowTaskInstanceParams failed")
	}
	pos := make([]*WorkflowTaskInstancePo, 0)
	if err := db.Find(&pos).Error; err != nil {
		return nil, errors.WithMessage(err, "QueryWorkflowTaskInstance failed")
	}
	return pos, nil
}

func buidUpdateWorkflowInstanceParams(db *gorm.DB, param *UpdateWorkflowInstanceParams) (*gorm.DB, error) {
	isHasWhere := false
	if param == nil {
		return nil, errors.New("nil UpdateWorkflowInstanceParams")
	}
	if param.Where == nil {
		return nil, errors.New("where is nil")
	}
	if param.Fields == nil {
		return nil, errors.New("fields is nil")
	}
	if len(param.Where.IDIn) > 0 {
		isHasWhere = true
		db = db.Where("id IN ?", param.Where.IDIn)
	}
	if len(param.Where.StatusIn) > 0 {
		isHasWhere = true
		db = db.Where("status IN ?", param.Where.StatusIn)
	}
	if !isHasWhere {
		return db, errors.New("update workflow instance need where condition, please check, params is %s")
	}
	return db, nil
}

func buildUpdateWorkflowInstanceFields(fields *UpdateWorkflowInstanceField) (map[string]any, error) {
	updateFields := make(map[string]interface{})
	if fields.Status != nil {
		updateFields["status"] = *fields.Status
	}
	if fields.WorkflowContext != nil {
		jsonData, err := fields.WorkflowContext.ToBytes()
		if err != nil {
			return nil, errors.WithMessage(err, "Marshal fields.Context failed")
		}
		updateFields["context"] = jsonData
	}
	if len(updateFields) == 0 {
		return nil, errors.New("no fields to update")
	}
	updateFields["updated_at"] = time.Now().Unix()
	return updateFields, nil
}
func (r *workflowRepo) UpdateWorkflowInstance(ctx context.Context, param *UpdateWorkflowInstanceParams) error {
	if param == nil {
		return fmt.Errorf("nil UpdateWorkflowInstanceParams")
	}
	db := r.GetDBWithContext(ctx).Model(&WorkflowInstancePo{})
	db, err := buidUpdateWorkflowInstanceParams(db, param)
	if err != nil {
		return errors.WithMessage(err, "buildUpdateWorkflowInstanceParams failed")
	}
	updateFields, err := buildUpdateWorkflowInstanceFields(param.Fields)
	if err != nil {
		return errors.WithMessage(err, "buildUpdateWorkflowInstanceFields failed")
	}
	if err := db.Updates(updateFields).Limit(param.LimitMax).Error; err != nil {
		return errors.WithMessage(err, "UpdateWorkflowInstance failed")
	}
	return nil
}

func buildUpdateWorkflowTaskInstanceParams(db *gorm.DB, param *UpdateWorkflowTaskInstanceParams) (*gorm.DB, error) {
	isHasWhere := false
	if param == nil {
		return nil, errors.New("nil UpdateWorkflowTaskInstanceParams")
	}
	if param.Where == nil {
		return nil, errors.New("where is nil")
	}
	if len(param.Where.IDIn) > 0 {
		isHasWhere = true
		db = db.Where("id IN ?", param.Where.IDIn)
	}
	if !isHasWhere {
		return db, errors.New("update workflow task instance need where condition, please check, params is %s")
	}
	return db, nil
}

func buildUpdateWorkflowTaskInstanceFields(fields *UpdateWorkflowTaskInstanceField) (map[string]interface{}, error) {
	updateFields := make(map[string]interface{})
	if fields.Status != nil {
		updateFields["status"] = *fields.Status
	}
	if fields.NodeContext != nil {
		jsonData, err := fields.NodeContext.ToBytes()
		if err != nil {
			return nil, errors.WithMessage(err, "Marshal fields.NodeContext failed")
		}
		updateFields["node_context"] = jsonData
	}
	if fields.FailCount != nil {
		updateFields["fail_count"] = *fields.FailCount
	}
	if len(updateFields) == 0 {
		return nil, errors.New("no fields to update")
	}

	updateFields["updated_at"] = time.Now().Unix()
	return updateFields, nil
}
func (r *workflowRepo) UpdateWorkflowTaskInstance(ctx context.Context, param *UpdateWorkflowTaskInstanceParams) error {
	if param == nil {
		return fmt.Errorf("nil UpdateWorkflowTaskInstanceParams")
	}
	db := r.GetDBWithContext(ctx).Model(&WorkflowTaskInstancePo{})
	db, err := buildUpdateWorkflowTaskInstanceParams(db, param)
	if err != nil {
		return errors.WithMessage(err, "buildUpdateWorkflowTaskInstanceParams failed")
	}
	updateFields, err := buildUpdateWorkflowTaskInstanceFields(param.Fields)
	if err != nil {
		return errors.WithMessage(err, "buildUpdateWorkflowTaskInstanceFields failed")
	}
	if err := db.Updates(updateFields).Limit(param.LimitMax).Error; err != nil {
		return errors.WithMessage(err, "UpdateWorkflowTaskInstance failed")
	}
	return nil
}

type contextKey string

const (
	transactionContextKey contextKey = "transaction"
)

func (r *workflowRepo) GetDBWithContext(ctx context.Context) *gorm.DB {
	tx := ctx.Value(transactionContextKey)
	if tx == nil {
		// 没有事务，直接返回mysql即可
		return r.db.WithContext(ctx)
	}
	return tx.(*gorm.DB)
}
func (r *workflowRepo) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	ctxTX := ctx.Value(transactionContextKey)
	var err error
	if ctxTX == nil {
		tx := r.db.Begin()
		defer func() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()
		newCtx := context.WithValue(ctx, transactionContextKey, tx)
		err = fn(newCtx)
		return err
	}
	return fn(ctx)
}
