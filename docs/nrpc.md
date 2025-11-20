---
title: NRPC v2 - 内置消息系统
description: 基于嵌入式NATS的高性能消息传递框架，支持服务间通信、实时数据流和分布式事件处理
order: 20
section: core-concepts
tags: [nrpc, nats, messaging, rpc, streaming, embedded]
category: docs
---

# NRPC v2 - 内置消息系统

NRPC v2 是基于嵌入式NATS构建的高性能消息传递框架，为MetaBase提供统一的服务间通信、实时数据流和分布式事件处理能力。

## 🚀 核心特性

### 嵌入式架构
- **零依赖部署**: 内置NATS服务器，无需外部依赖
- **自动生命周期管理**: 服务启动、停止和自动恢复
- **资源优化**: 内存使用和连接数自动优化

### 消息模式
- **请求-响应**: 同步RPC调用，支持超时和重试
- **发布-订阅**: 异步消息分发，支持多订阅者
- **流式传输**: 大数据量的分块传输和实时流处理
- **队列组**: 工作队列和负载均衡

### 高级特性
- **中间件支持**: 日志、认证、限流、熔断器
- **自动重连**: 网络断开时的自动重连和消息恢复
- **JetStream集成**: 持久化消息流和消息重放
- **多租户隔离**: 命名空间隔离和权限控制

## 🏗️ 架构设计

### 消息类型

```go
type MessageType string

const (
    MessageTypeRequest  MessageType = "request"  // 请求消息
    MessageTypeResponse MessageType = "response" // 响应消息
    MessageTypeError    MessageType = "error"    // 错误消息
    MessageTypeEvent    MessageType = "event"    // 事件消息
    MessageTypeStream   MessageType = "stream"   // 流式消息
    MessageTypePing     MessageType = "ping"     // 心跳消息
    MessageTypePong     MessageType = "pong"     // 心跳响应
)
```

## 📝 快速开始

### 1. 创建 NRPC 服务器

```go
// 创建嵌入式 NATS
natsConfig := &embedded.Config{
    ServerPort: 4222,
    ClientURL:  "nats://localhost:4222",
    StoreDir:   "./data/nats",
    JetStream:  true,
}

nats := embedded.NewEmbeddedNATS(natsConfig)
if err := nats.Start(); err != nil {
    log.Fatal("Failed to start NATS:", err)
}

// 创建 NRPC 服务器
nrpcConfig := &nrpc.Config{
    Name:            "my-service",
    Version:         "1.0.0",
    Namespace:       "myapp",
    EnableStreaming: true,
    EnableMetrics:   true,
}

server := nrpc.NewServer(nats, nrpcConfig)

// 添加中间件
server.Use(middleware.NewLoggingMiddleware(log.Default()))
server.Use(middleware.NewMetricsMiddleware())

// 启动服务器
if err := server.Start(); err != nil {
    log.Fatal("Failed to start NRPC server:", err)
}
```

### 2. 实现服务处理器

```go
type UserService struct{}

func NewUserService() *UserService {
    builder := nrpc.NewServiceBuilder("user")

    // 注册获取用户方法
    builder.Method("get", "Get user by ID", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
        userID, ok := req.Data["user_id"].(string)
        if !ok {
            return nil, fmt.Errorf("user_id required")
        }

        user := map[string]interface{}{
            "id":    userID,
            "name":  "John Doe",
            "email": "john@example.com",
        }

        return &nrpc.Response{
            ID:   req.ID,
            Data: user,
        }, nil
    })

    return builder.Build()
}
```

### 3. 创建客户端

```go
// 创建客户端
clientConfig := &nrpc.ClientConfig{
    Namespace: "myapp",
    Timeout:   10 * time.Second,
}

client := nrpc.NewClient(nats, clientConfig)

// 同步调用
ctx := context.Background()
response, err := client.Call(ctx, "user", "get", map[string]interface{}{
    "user_id": "user_123",
}, nil)

// 发布事件
err = client.Publish(ctx, "user.created", map[string]interface{}{
    "user_id": "user_456",
}, nil)

// 订阅事件
subscription, err := client.Subscribe("user.*", func(msg *nrpc.Message) {
    fmt.Printf("Received event: %+v\n", msg)
})
```

## 🔧 中间件系统

### 内置中间件

```go
// 日志中间件
server.Use(middleware.NewLoggingMiddleware(log.Default()))

// 认证中间件
server.Use(middleware.NewAuthMiddleware(func(token string) (map[string]interface{}, error) {
    return validateJWTToken(token)
}))

// 限流中间件
server.Use(middleware.NewRateLimitMiddleware(100, time.Minute))

// 熔断器中间件
server.Use(middleware.NewCircuitBreakerMiddleware(5, time.Minute))
```

### 自定义中间件

```go
type CustomMiddleware struct{}

func (cm *CustomMiddleware) Handle(ctx context.Context, req *nrpc.Request, next nrpc.NextFunc) (*nrpc.Response, error) {
    // 前置处理
    start := time.Now()

    // 调用下一个中间件或服务
    resp, err := next(ctx, req)

    // 后置处理
    duration := time.Since(start)
    log.Printf("Request %s.%s took %v", req.Service, req.Method, duration)

    return resp, err
}
```

## 🚀 高级用法

### 流式数据处理

```go
// 服务端流式处理
builder.StreamingMethod("process_data", "Process large dataset", func(ctx context.Context, req *nrpc.Request) (*nrpc.Response, error) {
    for i := 0; i < 1000; i++ {
        data := processBatch(i)
        // 通过流发送数据
        // stream.Send(data)
    }

    return &nrpc.Response{
        ID:   req.ID,
        Data: map[string]interface{}{"processed": 1000},
    }, nil
})

// 客户端接收流
stream, err := client.Stream(ctx, "data", "process_data", map[string]interface{}{
    "source": "large_dataset.csv",
}, nil)

for msg := range stream {
    fmt.Printf("Stream data: %+v\n", msg.Data)
    if msg.StreamEnd {
        break
    }
}
```

### 事件驱动架构

```go
// 发布用户事件
func publishUserEvent(userID, eventType string, data map[string]interface{}) error {
    event := map[string]interface{}{
        "user_id":    userID,
        "event_type": eventType,
        "data":       data,
        "timestamp":  time.Now().Unix(),
    }

    return client.Publish(context.Background(), "user.events", event, nil)
}

// 订阅用户事件
func subscribeToUserEvents() {
    subscription, _ := client.Subscribe("user.events.*", func(msg *nrpc.Message) {
        eventType := strings.TrimPrefix(msg.Subject, "user.events.")
        userID := msg.Data["user_id"].(string)

        switch eventType {
        case "created":
            handleUserCreated(userID, msg.Data)
        case "updated":
            handleUserUpdated(userID, msg.Data)
        }
    })
}
```

## 📊 监控和指标

### 健康检查

```go
// 客户端健康检查
health, err := client.HealthCheck(ctx)
if err != nil {
    log.Fatal("Health check failed:", err)
}

fmt.Printf("Server status: %v\n", health["status"])
fmt.Printf("NATS ready: %v\n", health["nats_ready"])
```

### 性能指标

```go
// 获取服务器信息
info, err := client.GetInfo(ctx)
fmt.Printf("Server: %s v%s\n", info["name"], info["version"])
fmt.Printf("Services: %v\n", info["services"])
```

## 🔒 安全特性

### 认证和授权

```go
// JWT 认证中间件
func authMiddleware(token string) (map[string]interface{}, error) {
    claims, err := validateJWT(token)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    if !hasPermission(claims, "service", "access") {
        return nil, fmt.Errorf("insufficient permissions")
    }

    return claims, nil
}

server.Use(middleware.NewAuthMiddleware(authMiddleware))
```

## 🚀 部署指南

### 单机部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  metabase:
    build: .
    ports:
      - "7609:7609"   # HTTP API
      - "4222:4222"   # NATS
      - "8222:8222"   # NATS Monitoring
    environment:
      - METABASE_NATS_PORT=4222
      - METABASE_NATS_STORE_DIR=/data/nats
    volumes:
      - ./data:/data
```

### 集群部署

```go
// 集群配置
natsConfig := &embedded.Config{
    ServerPort: 4222,
    Cluster: &embedded.ClusterConfig{
        Name: "metabase-cluster",
        Routes: []string{
            "nats://node1:6222",
            "nats://node2:6222",
            "nats://node3:6222",
        },
    },
    JetStream: true,
}
```

## 🎯 最佳实践

### 1. 服务设计
- **单一职责**: 每个服务专注于一个业务领域
- **无状态**: 服务本身不保存状态，状态存储在外部
- **幂等性**: 确保重复调用产生相同结果
- **版本化**: 使用版本号管理API变更

### 2. 错误处理
- **结构化错误**: 使用标准错误格式
- **错误传播**: 在调用链中正确传播错误
- **重试策略**: 实现指数退避重试
- **熔断机制**: 防止级联故障

### 3. 性能优化
- **连接池**: 复用NATS连接
- **批处理**: 批量处理小消息
- **流式处理**: 大数据量使用流式传输
- **缓存**: 缓存频繁访问的数据

NRPC v2 为MetaBase提供了强大而灵活的消息传递基础设施，支持从简单的RPC调用到复杂的分布式事件处理等各种场景。