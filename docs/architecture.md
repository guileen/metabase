---
title: 架构
description: MetaBase 的核心架构设计，包括嵌入式NATS消息队列、存储引擎、文件系统和分析控制台四大核心组件。
order: 10
section: core-concepts
tags: [architecture, design, overview]
category: docs
---

# 架构

MetaBase 采用一体化架构设计，核心由四部分组成：

- **嵌入式 NATS**：内置消息队列与 RPC，无需外部依赖，支持重试与延迟队列。
- **混合存储引擎**：SQLite + Pebble 组合，持久化与索引分离，内置文件系统。
- **实时分析引擎**：访问日志、搜索、仪表盘与报表生成一体化。
- **现代控制台**：Svelte 构建的管理界面，统一管理与监控。

所有模块以异步为先，接口保持无状态与可测试性。

## 核心组件

### 1. 嵌入式 NATS 消息队列

MetaBase 内置嵌入式 NATS 服务器，提供：

```go
// 嵌入式配置
natsConfig := &nrpc.Config{
    Embedded: true,           // 嵌入式模式
    DataDir:  "./data/nats",   // 数据持久化目录
    Port:     4222,          // 端口
    Cluster:  false,         // 单机模式
}

// 自动启动，无需外部依赖
server, _ := nrpc.NewServer(natsConfig)
```

**优势：**
- ✅ **零配置**：无需安装外部 NATS 服务器
- ✅ **高性能**：内存中运行，延迟微秒级
- ✅ **持久化**：消息自动持久化到磁盘
- ✅ **可靠性**：支持重试、死信队列、延迟消息

### 2. 混合存储引擎

采用 SQLite + Pebble 的混合架构：

```go
// 存储配置
storageConfig := &storage.Config{
    SQLitePath:   "./data/metabase.db",  // 主数据库
    PebblePath:   "./data/pebble",       // 索引和缓存
    CacheEnabled: true,
    CacheSize:    1000,
}

// 自动初始化
engine, _ := storage.NewEngine(storageConfig)
```

**架构特点：**
- **SQLite**：存储主要数据，支持复杂查询和事务
- **Pebble**：高性能 KV 存储，用于索引、缓存、会话数据
- **文件系统**：内置文件存储，支持本地、S3、MinIO 等后端

### 3. 实时分析引擎

完整的访问分析和监控体系：

```go
// 分析配置
analyticsConfig := &analytics.Config{
    Storage:          storageConfig,
    RetentionPeriod:  90 * 24 * time.Hour, // 90天数据保留
    EnableRealTime:   true,
    MaxResults:       10000,
}

// 创建分析引擎
analytics, _ := analytics.NewEngine(analyticsConfig)

// 实时事件跟踪
analytics.TrackEvent(ctx, &analytics.Event{
    Type:      analytics.EventTypeAPIRequest,
    TenantID:  "tenant123",
    UserID:    "user456",
    EventName: "api_call",
    Properties: map[string]interface{}{
        "method": "POST",
        "path":   "/api/users",
        "status": 200,
    },
})
```

**功能特性：**
- **实时统计**：QPS、响应时间、错误率
- **事件跟踪**：页面访问、API 调用、用户行为
- **搜索分析**：全文搜索、分面分析
- **仪表盘**：自定义图表和布局
- **报表系统**：定时生成、多格式导出

### 4. 文件存储系统

企业级文件管理能力：

```go
// 文件配置
fileConfig := &files.Config{
    StorageType:    files.StorageLocal,
    LocalPath:      "./uploads",
    MaxFileSize:    100 * 1024 * 1024, // 100MB
    EnableDedup:    true,
    Compression: &files.CompressionConfig{
        Enabled:   true,
        Algorithm: "gzip",
        Level:     6,
    },
}

// 创建文件引擎
files, _ := files.NewEngine(fileConfig, storageEngine)

// 上传文件
metadata, _ := files.Upload(ctx, file, header, &files.UploadOptions{
    TenantID:  "tenant123",
    IsPublic:  false,
    Tags:      []string{"profile", "image"},
})
```

**存储后端支持：**
- **本地存储**：直接文件系统存储
- **S3**：AWS S3 兼容存储
- **MinIO**：私有对象存储
- **分布式文件系统**：可插拔的分布式存储

## 服务架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    MetaBase Server                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │    HTTP     │  │  Embedded   │  │   Analytics │          │
│  │   Router    │  │    NATS     │  │   Engine    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Storage   │  │    Files    │  │    Auth     │          │
│  │   Engine    │  │   Storage   │  │  Service    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Tenant    │  │     RLS     │  │   Session   │          │
│  │  Manager    │  │   Engine    │  │  Manager    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Data Layer                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   SQLite    │  │   Pebble    │  │  File System│          │
│  │  Database   │  │     KV      │  │   Storage   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

## 数据流架构

```
Client Request → HTTP Router → Authentication → Authorization → Business Logic
                                    │                     │
                                    ▼                     ▼
                               Analytics              Storage Engine
                                    │                     │
                                    ▼                     ▼
                              Event Tracking        SQLite + Pebble
                                    │                     │
                                    ▼                     ▼
                              Real-time Stats         File Storage
                                    │                     │
                                    ▼                     ▼
                              Dashboard          File Management
```

## 部署模式

### 单机模式 (推荐)
```bash
# 一键启动，所有服务内嵌
metabase server --enable-all

# 开发模式，热重载
make dev-full
```

### 分布式模式
```bash
# 数据存储节点
metabase server --role=storage --data-dir=/data

# API 服务节点
metabase server --role=api --storage-url=storage-node:7609

# 分析节点
metabase server --role=analytics --storage-url=storage-node:7609
```

## 性能特性

### 高并发处理
- **异步处理**：所有 I/O 操作异步化
- **连接池**：数据库连接复用
- **批处理**：批量写入优化
- **缓存策略**：多层缓存机制

### 内存优化
- **对象池**：减少 GC 压力
- **流式处理**：大数据集流式读取
- **延迟加载**：按需加载机制

### 存储优化
- **索引分离**：热数据 Pebble，冷数据 SQLite
- **压缩存储**：自动数据压缩
- **分片存储**：大文件分片处理

## 安全架构

### 多租户隔离
- **数据隔离**：租户级别数据隔离
- **资源隔离**：CPU、内存、存储配额
- **网络隔离**：租户间网络隔离

### 访问控制
- **RBAC**：基于角色的访问控制
- **RLS**：行级安全策略
- **API 密钥**：安全的 API 访问认证

### 数据保护
- **加密存储**：敏感数据加密
- **传输加密**：HTTPS/TLS 加密
- **审计日志**：完整的操作审计

## 扩展性设计

### 水平扩展
- **无状态设计**：服务层无状态
- **负载均衡**：支持负载均衡
- **数据分片**：支持数据水平分片

### 插件化
- **存储插件**：可插拔存储后端
- **认证插件**：可插拔认证方式
- **分析插件**：可扩展分析功能

### API 设计
- **RESTful API**：标准 REST 接口
- **GraphQL**：灵活的查询语言
- **WebSocket**：实时通信支持

## 开发体验

### 热重载
```bash
# 代码修改自动重启
make dev

# 前端热重载
make admin-dev
```

### 测试支持
```bash
# 单元测试
make test

# 集成测试
make test-integration

# 性能测试
make benchmark
```

### 调试工具
- **健康检查**：`/api/health`
- **指标监控**：`/api/stats`
- **调试模式**：`--dev` 参数
- **日志查看**：结构化日志输出

这种一体化架构设计让 MetaBase 能够在保持简单性的同时提供企业级功能，真正实现"开箱即用"的后端核心。