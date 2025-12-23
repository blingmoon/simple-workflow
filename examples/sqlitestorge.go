package examples

import (
	"github.com/blingmoon/simple-workflow/workflow"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSqliteStore() (workflow.WorkflowService, error) {
	db, err := gorm.Open(sqlite.Open("test.sqlite3"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&workflow.WorkflowInstancePo{}, &workflow.WorkflowTaskInstancePo{})
	workflowRepo := workflow.NewWorkflowRepo(db)
	return workflow.NewWorkflowService(workflowRepo, workflow.NewLocalWorkflowLock()), nil
}
