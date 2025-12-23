# 贡献指南

感谢你考虑为本项目做出贡献！

## 如何贡献

### 报告 Bug

如果你发现了 bug，请创建一个 issue，并包含以下信息：

- 清晰的标题和描述
- 重现步骤
- 预期行为和实际行为
- Go 版本和操作系统信息
- 相关的代码片段或日志

### 提交代码

1. Fork 本仓库
2. 创建你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

### 代码规范

- 遵循 Go 官方的代码风格指南
- 运行 `go fmt` 格式化代码
- 运行 `go vet` 检查代码
- 确保所有测试通过 (`go test ./...`)
- 为新功能添加测试
- 更新相关文档

## 开发流程

### 环境设置

```bash
# 克隆仓库
git clone https://github.com/yourusername/simple-workflow.git
cd simple-workflow

# 运行测试
go test ./...
```

### 提交信息规范

使用清晰的提交信息：

```
feat: 添加新功能
fix: 修复 bug
docs: 更新文档
test: 添加测试
refactor: 重构代码
```

## 许可证

贡献的代码将遵循项目的 MIT 许可证。

