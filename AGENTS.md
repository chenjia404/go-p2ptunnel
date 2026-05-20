## Cursor Cloud specific instructions

### 项目概述

go-p2ptunnel 是一个基于 libp2p 的 P2P TCP 隧道工具（Go 语言），支持 QUIC/TCP/WebSocket/WebTransport 传输协议，使用 Noise 协议加密，内置 NAT 穿越、SOCKS5 代理和 HTTP/HTTPS 代理功能。

这是一个单体 Go 项目，无外部服务依赖（无数据库、无消息队列）。

### 开发命令

- **构建**: `go build -trimpath -ldflags="-w -s"`
- **测试**: `go test ./...`
- **静态检查**: `go vet ./...`（注意：当前代码库有已知的 vet 警告，非阻塞性问题）
- **下载依赖**: `go mod download`

### 运行说明

应用使用 fork-daemon 模式运行。设置环境变量 `__NewProc=true` 可跳过 daemon 模式直接运行子进程逻辑，便于调试和测试。

运行两个实例进行端到端测试时：
1. 服务端：`__NewProc=true ./go-p2ptunnel -l 127.0.0.1:<目标端口> -nodisc -p2p_port 4001 -user server`
2. 从服务端输出中获取节点 ID
3. 客户端：`__NewProc=true ./go-p2ptunnel -id /ip4/127.0.0.1/tcp/4001/p2p/<服务端ID> -l 127.0.0.1:<本地端口> -nodisc -p2p_port 4003 -user client`

每个实例需要独立的工作目录（用于存放 `.key` 文件），且 `-p2p_port` 不能冲突。

### 注意事项

- QUIC UDP 缓冲区大小警告（`failed to sufficiently increase receive buffer size`）不影响功能，仅影响性能。
- 使用 `-nodisc` 标志关闭节点发现可加速本地测试。
- 项目要求 Go 1.26+（见 `go.mod`）。
