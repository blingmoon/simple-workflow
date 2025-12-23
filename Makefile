.PHONY: test coverage fmt vet tidy help

test: ## 运行测试
	go test -v ./...

coverage: ## 生成测试覆盖率报告
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt: ## 格式化代码
	go fmt ./...

vet: ## 运行 go vet
	go vet ./...

tidy: ## 整理依赖
	go mod tidy

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

