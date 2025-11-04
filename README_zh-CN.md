<div align="center">
  <img src="https://github.com/DeepLink-org/DeepTrace/releases/download/v0.1.0-beta/deeptrace.png" width="450"/>

   [![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

   [English](README.md) | 简体中文
</div>

# DeepTrace

## 项目简介

DeepTrace是一种分布式训练中任务排查、诊断的实现方案。它采用client-agent的结构设计，agent轻量级部署，对训练任务侵入性小。client和agent使用gRPC协议，支持实时数据推送和命令控制。分布式设计适合大规模训练集群需求，具有很好的扩展性，后续可继续添加新的排障和分析功能。

## 功能特性

- **分布式架构**：采用client-agent结构，适合大规模训练集群
- **轻量级Agent**：对训练任务侵入性小，易于部署
- **实时数据推送**：基于gRPC协议，支持实时数据传输
- **多维度诊断**：支持日志分析和堆栈跟踪
- **可扩展设计**：模块化架构，易于添加新功能
- **命令行接口**：提供CLI工具，便于运维人员使用

## 架构设计

### 架构说明：

1. 客户端组件
   - CLI命令行接口：运维人员使用的命令行工具
   - 分析引擎：核心分析逻辑，处理从Agent收集的数据
2. 训练集群层
   - 诊断Agent：监控训练任务（by 日志、堆栈）
3. 通信协议
   - 使用gRPC协议

## 安装指南

### 环境要求

- Go 1.24+

### 构建

```bash
make build
```

构建后的二进制文件将在Makgefile中指定的版本信息。

### 修改protobuf定义

修改protobuf定义后，需要执行以下命令生成代码：

```bash
make generate
```

## 使用说明

### 客户端命令

执行hang检测：
```bash
client check-hang --job-id my_job -w clusterx --threshold 120 --interval 5
```

## API文档

详细API文档请参考[proto文件](v1/deeptrace.proto)。

## 贡献指南

欢迎贡献代码！在提交Pull Request之前，请确保：

1. 编写测试用例
2. 更新相关文档
3. 确保所有测试通过

### 开发环境设置

1. Fork项目
2. 克隆到本地
3. 创建功能分支
4. 提交更改
5. 推送到分支
6. 创建Pull Request

## 许可证

本项目采用Apache License 2.0许可证，详情请见[LICENSE](LICENSE)文件。

## 联系方式

如有问题，请提交Issue或联系项目维护者。