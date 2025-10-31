<div align="center">
  <img src="https://github.com/DeepLink-org/DeepTrace/releases/download/v0.1.0-beta/deeptrace.png" width="450"/>

   [![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

   English | [简体中文](README_zh-CN.md)
</div>


# DeepTrace

## Project Overview

DeepTrace is a distributed training task diagnostic solution. It adopts a client-agent architecture with lightweight agent deployment that minimally intrudes on training tasks. The client and agent communicate via gRPC protocol, supporting real-time data streaming and command control. The distributed design caters to large-scale training cluster requirements with excellent scalability, allowing future expansion of new troubleshooting and analysis features.

## Key Features

- **Distributed Architecture**: Client-agent structure suitable for large training clusters
- **Lightweight Agent**: Minimal intrusion on training tasks, easy deployment
- **Real-time Data Streaming**: gRPC-based protocol supports real-time data transmission
- **Multi-dimensional Diagnostics**: Supports log analysis and stack tracing
- **Extensible Design**: Modular architecture for easy feature additions
- **CLI Interface**: Provides command-line tools for operational maintenance

## Architecture Design

### Architecture Components:

1. Client Components
   - CLI Interface: Maintenance personnel's command-line tool
   - Analysis Engine: Core logic processing data from Agents
2. Training Cluster Layer
   - Diagnostic Agent: Monitors training tasks (via logs/stacks)
3. Communication Protocol
   - Uses gRPC protocol

## Installation Guide

### Requirements

- Go 1.24+

### Build

```bash
make build
```

Built binaries will contain version information specified in Makefile.

### Protobuf Generation

After modifying protobuf definitions, run:

```bash
make generate
```

## Usage

### Client Commands

Execute hang detection:

```bash
client check-hang --job-id my_job -w clusterx --threshold 120 --interval 5
```

## API Documentation

Detailed API reference see [proto file](v1/deeptrace.proto).

## Contributing

We welcome contributions! Before submitting a PR:

1. Write test cases
2. Update relevant documentation
3. Ensure all tests pass

### Development Setup

1. Fork project
2. Clone repository
3. Create feature branch
4. Commit changes
5. Push to branch
6. Create Pull Request

## License

Apache License 2.0 - See [LICENSE](LICENSE).

## Contact

Please submit issues or contact maintainers for support.