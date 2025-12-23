# Simple Workflow

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/blingmoon/simple-workflow)](https://goreportcard.com/report/github.com/blingmoon/simple-workflow)
[![GoDoc](https://godoc.org/github.com/blingmoon/simple-workflow?status.svg)](https://godoc.org/github.com/blingmoon/simple-workflow)

一个简单的 Go 工作流编排库。

## 特性

- 🚀 简单易用的 API 设计
- 📦 可直接导入使用
- 🧪 完善的测试覆盖
- 📝 清晰的文档

## 安装

```bash
go get github.com/blingmoon/simple-workflow
```

## 快速开始

```go
package main

import (
    workflow "github.com/blingmoon/simple-workflow"
)

func main() {
    // 创建工作流
    wf := workflow.New("my-workflow")
    
    // 执行工作流
    if err := wf.Run(); err != nil {
        panic(err)
    }
}
```

## 示例

查看 [examples](examples/) 目录获取更多示例。

## 文档

完整的 API 文档请访问 [GoDoc](https://godoc.org/github.com/blingmoon/simple-workflow)。

## 测试

```bash
# 运行测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...
```

## 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 联系方式

- Issues: [GitHub Issues](https://github.com/blingmoon/simple-workflow/issues)
- Discussions: [GitHub Discussions](https://github.com/blingmoon/simple-workflow/discussions)
