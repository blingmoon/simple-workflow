# CI/CD 配置说明

## 📋 概述

项目使用 GitHub Actions 进行持续集成，包含测试和代码质量检查。

## 🔧 CI 工作流程

### 1. Test Job（测试任务）

测试任务运行在 `ubuntu-latest` 上，使用 Go 1.24。

#### 测试步骤：

1. **下载依赖**
   ```bash
   go mod download          # 主项目依赖
   cd tests && go mod download  # 测试模块依赖
   ```

2. **运行 workflow 包测试**
   ```bash
   go test -v -race -coverprofile=coverage-workflow.out ./workflow/...
   ```
   - 测试 `workflow/` 包内部的单元测试
   - 生成 `coverage-workflow.out`

3. **运行 tests 目录测试**
   ```bash
   cd tests
   go test -v -race -coverprofile=coverage.out \
     -coverpkg=github.com/blingmoon/simple-workflow/workflow ./...
   ```
   - 测试 `tests/` 目录的集成测试
   - 计算对 `workflow` 包的覆盖率
   - 生成 `tests/coverage.out`

4. **合并覆盖率报告**
   ```bash
   gocovmerge coverage-workflow.out tests/coverage.out > coverage-merged.out
   ```
   - 使用 `gocovmerge` 合并两个覆盖率文件

5. **显示覆盖率摘要**
   - Workflow 包内部测试覆盖率
   - Tests 目录测试覆盖率
   - 合并后总覆盖率
   - 输出到 GitHub Step Summary

6. **上传到 Codecov**
   - 上传合并后的覆盖率报告
   - 需要配置 `CODECOV_TOKEN` secret

### 2. Lint Job（代码检查任务）

使用 `golangci-lint` 进行代码质量检查。

- **版本**: golangci-lint v6（最新版）
- **Go 版本**: 1.24
- **失败处理**: `continue-on-error: true`（不会阻塞 CI）

## 📊 覆盖率统计

### 覆盖率来源

| 来源 | 说明 | 覆盖率文件 |
|------|------|-----------|
| workflow 包内测试 | `workflow/json_context_test.go` | `coverage-workflow.out` |
| tests 目录测试 | `tests/workflow_*_test.go`, `tests/integration_test.go` | `tests/coverage.out` |
| **合并后** | 综合覆盖率 | `coverage-merged.out` |

### 预期覆盖率

- workflow 包内测试：~6%
- tests 目录测试：~54%
- **合并后总覆盖率：~55%+**

## 🚀 本地测试

### 方法 1：使用脚本（推荐）

```bash
# 运行完整的 CI 测试流程
.vscode/.ai/test_ci_locally.sh
```

这个脚本会：
1. ✅ 下载所有依赖
2. ✅ 运行所有测试
3. ✅ 生成覆盖率报告
4. ✅ 合并覆盖率数据
5. ✅ 生成 HTML 报告
6. ✅ 运行 lint 检查（如果安装）

### 方法 2：手动运行

```bash
# 1. 下载依赖
go mod download
cd tests && go mod download && cd ..

# 2. 运行 workflow 包测试
go test -v -race -coverprofile=coverage-workflow.out ./workflow/...

# 3. 运行 tests 目录测试
cd tests
go test -v -race -coverprofile=coverage.out \
  -coverpkg=github.com/blingmoon/simple-workflow/workflow ./...
cd ..

# 4. 安装 gocovmerge（首次需要）
go install github.com/wadey/gocovmerge@latest

# 5. 合并覆盖率
gocovmerge coverage-workflow.out tests/coverage.out > coverage-merged.out

# 6. 查看覆盖率
go tool cover -func=coverage-merged.out | tail -1

# 7. 生成 HTML 报告
go tool cover -html=coverage-merged.out -o coverage.html
open coverage.html
```

## 🔑 必需的 Secrets

在 GitHub 仓库设置中配置：

### CODECOV_TOKEN（可选）

如果使用 Codecov 上传覆盖率：

1. 访问 https://codecov.io/
2. 登录并添加你的仓库
3. 复制 Upload Token
4. 在 GitHub Settings → Secrets → Actions 中添加 `CODECOV_TOKEN`

**不配置的影响**：覆盖率不会上传到 Codecov，但测试仍会正常运行。

## 📁 相关文件

```
.github/
├── workflows/
│   ├── ci.yml          # CI 配置文件
│   └── CI_SETUP.md     # 本文件
.vscode/.ai/
└── test_ci_locally.sh  # 本地测试脚本

tests/
├── go.mod              # 独立的测试模块
├── workflow_basic_test.go
├── workflow_advanced_test.go
├── integration_test.go
└── json_context_simple_test.go

workflow/
└── json_context_test.go  # workflow 包内部测试

go.work                  # Go workspace 配置
```

## 🔄 CI 触发条件

### Push 事件
- `master` 分支
- `develop` 分支

### Pull Request 事件
- 目标分支为 `master`
- 目标分支为 `develop`

## 📈 查看 CI 结果

### GitHub Actions

1. 访问仓库的 Actions 页面
2. 选择最新的 workflow run
3. 查看 Test job 的输出
4. 在 Summary 中查看覆盖率摘要

### Codecov（如果配置）

访问 https://codecov.io/gh/YOUR_USERNAME/simple-workflow

## 🐛 常见问题

### Q1: gocovmerge 命令找不到

**问题**: CI 中 `gocovmerge: command not found`

**解决**: 
```yaml
# 已在 ci.yml 中添加
go install github.com/wadey/gocovmerge@latest
```

### Q2: 覆盖率计算不准确

**问题**: 覆盖率数字不对

**原因**: 
- 可能只运行了部分测试
- 可能 `-coverpkg` 参数不正确

**解决**: 
```bash
# 确保指定正确的 coverpkg
cd tests
go test -coverpkg=github.com/blingmoon/simple-workflow/workflow ./...
```

### Q3: tests 目录找不到 workflow 包

**问题**: `could not import github.com/blingmoon/simple-workflow/workflow`

**原因**: 
- `go.work` 未正确配置
- 或在 CI 中被忽略

**解决**: 
确保 `go.work` 文件存在且包含：
```go
use (
    .
    ./tests
)
```

### Q4: Lint 失败导致 CI 失败

**问题**: golangci-lint 报错导致整个 CI 失败

**解决**: 
已在 ci.yml 中添加 `continue-on-error: true`，lint 失败不会阻塞 CI。

## 🔧 自定义配置

### 修改 Go 版本

在 `.github/workflows/ci.yml` 中：

```yaml
strategy:
  matrix:
    go-version: ['1.24']  # 修改这里
```

### 添加更多测试

1. 在 `tests/` 目录添加新的测试文件
2. 测试会自动被 CI 执行
3. 覆盖率会自动更新

### 修改覆盖率目标

可以添加覆盖率检查：

```yaml
- name: Check coverage threshold
  run: |
    coverage=$(go tool cover -func=coverage-merged.out | tail -1 | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 50" | bc -l) )); then
      echo "❌ Coverage $coverage% is below 50%"
      exit 1
    fi
    echo "✅ Coverage $coverage% meets threshold"
```

## 📚 相关文档

- [测试覆盖率报告](../tests/COVERAGE_IMPROVED.md)
- [测试快速开始](../tests/QUICKSTART.md)
- [测试文档](../tests/README.md)
- [Go Workspace 文档](https://go.dev/ref/mod#workspaces)

## 🎯 最佳实践

1. **提交前运行本地测试**
   ```bash
   .vscode/.ai/test_ci_locally.sh
   ```

2. **保持测试快速**
   - 使用内存数据库（SQLite `:memory:`）
   - 避免 sleep 等待

3. **保持覆盖率**
   - 目标：>50%
   - 新功能添加对应测试

4. **及时修复 Lint 警告**
   - 虽然不阻塞 CI，但应该修复

## 📞 需要帮助？

如果遇到问题：

1. 查看 [常见问题](#-常见问题)
2. 运行本地测试脚本验证
3. 检查 GitHub Actions 日志
4. 查看相关文档

---

**最后更新**: 2025-12-23  
**维护者**: Simple Workflow Team

