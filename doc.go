// Package workflow 提供工作流编排功能。
//
// 这个包可以帮助你构建和管理任务流程。
//
// 基础使用示例:
//
//	package main
//
//	import (
//	    workflow "github.com/blingmoon/simple-workflow"
//	)
//
//	func main() {
//	    wf := workflow.New("my-workflow")
//	    if err := wf.Run(); err != nil {
//	        panic(err)
//	    }
//	}
//
// 更多示例和文档请访问: https://github.com/blingmoon/simple-workflow
package workflow

