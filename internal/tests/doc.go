// Package tests 是 simple-workflow 的内部测试模块。
//
// ⚠️ 重要提示：此包位于 internal/ 目录下，受 Go 编译器保护，
// 外部项目无法导入（会得到编译错误）。
//
// 🔒 编译器保护
//
// 如果外部项目尝试导入：
//
//	import "github.com/blingmoon/simple-workflow/internal/tests"
//
// 将会得到编译错误：
//
//	use of internal package github.com/blingmoon/simple-workflow/internal/tests not allowed
//
// 📋 测试内容
//
// 此模块包含以下测试：
//   - workflow 包的单元测试
//   - workflow 包的集成测试
//   - JSONContext 功能测试
//   - 并发场景测试
//   - 错误处理测试
//
// 📊 测试覆盖率
//
// 当前测试覆盖率约为 54.5%，合并主模块测试后达到 57.2%。
//
// 🚀 运行测试
//
// 在项目根目录：
//
//	cd internal/tests
//	go test ./...
//
// 查看覆盖率：
//
//	go test -coverprofile=coverage.out -coverpkg=github.com/blingmoon/simple-workflow/workflow ./...
//	go tool cover -html=coverage.out
//
// 📚 更多信息
//
// 参考文档：
//   - README.md - 测试模块说明
//   - QUICKSTART.md - 快速开始
//   - COVERAGE_IMPROVED.md - 覆盖率报告
package tests

