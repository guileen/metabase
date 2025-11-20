---
title: API 参考
description: MetaBase 完整的 API 文档，包括存储、文件、分析和认证等所有接口。
order: 100
section: api
tags: [api, rest, reference]
category: docs
---

# API 参考

MetaBase 提供完整的 RESTful API，支持数据存储、文件管理、实时分析和用户认证。

## 🌐 HTTP 接口

### 基础信息

- **Base URL**: `http://localhost:7609`
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

### 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 📄 文档 API

### 获取文档列表

```http
GET /api/docs
```

**响应示例**:
```json
{
  "status": "success",
  "data": [
    {
      "title": "总览",
      "url": "/docs/overview",
      "section": "getting-started",
      "order": 10,
      "description": "MetaBase 是为一人公司与小团队打造的下一代后端核心"
    },
    {
      "title": "架构",
      "url": "/docs/architecture",
      "section": "core-concepts",
      "order": 10,
      "description": "核心由三部分组成：NRPC、存储引擎、控制台"
    }
  ]
}
```

### 获取文档内容

```http
GET /api/docs/{slug}
```

**路径参数**:
- `slug`: 文档标识符

**响应示例**:
```json
{
  "status": "success",
  "data": {
    "title": "总览",
    "description": "MetaBase 是为一人公司与小团队打造的下一代后端核心",
    "content": "<h1>总览</h1><p>MetaBase 是为一人公司与小团队打造的...</p>",
    "section": "getting-started",
    "order": 10,
    "tags": ["intro", "overview"],
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

## 🔍 搜索 API

### 搜索文档

```http
GET /api/search?q={query}&limit={limit}&section={section}
```

**查询参数**:
- `q` (必需): 搜索关键词
- `limit` (可选): 返回结果数量限制，默认 10
- `section` (可选): 限定搜索的分组

**响应示例**:
```json
{
  "status": "success",
  "query": "架构",
  "total": 2,
  "data": [
    {
      "title": "架构",
      "url": "/docs/architecture",
      "section": "core-concepts",
      "snippet": "核心由三部分组成：NRPC、存储引擎、控制台",
      "score": 0.95
    },
    {
      "title": "存储引擎",
      "url": "/docs/storage",
      "section": "core-concepts",
      "snippet": "存储引擎：Sqlite + Pebble 组合...",
      "score": 0.87
    }
  ]
}
```

## 📁 导航 API

### 获取导航结构

```http
GET /api/nav
```

**响应示例**:
```json
{
  "status": "success",
  "data": [
    {
      "title": "开始使用",
      "items": [
        {
          "title": "总览",
          "url": "/docs/overview",
          "order": 10,
          "active": false
        },
        {
          "title": "快速开始",
          "url": "/docs/start",
          "order": 20,
          "active": true
        }
      ]
    },
    {
      "title": "核心概念",
      "items": [
        {
          "title": "架构",
          "url": "/docs/architecture",
          "order": 10,
          "active": false
        }
      ]
    }
  ]
}
```

## 📊 统计 API

### 获取站点统计

```http
GET /api/stats
```

**响应示例**:
```json
{
  "status": "success",
  "data": {
    "total_docs": 15,
    "total_sections": 4,
    "last_updated": "2024-01-01T12:00:00Z",
    "version": "1.0.0"
  }
}
```

## 🔧 管理 API (开发模式)

### 重新扫描文档

```http
POST /api/admin/rescan
```

**响应示例**:
```json
{
  "status": "success",
  "message": "文档扫描完成",
  "scanned": 15,
  "updated": 2
}
```

### 清除缓存

```http
POST /api/admin/cache/clear
```

**响应示例**:
```json
{
  "status": "success",
  "message": "缓存已清除"
}
```

## 🚨 错误处理

### 标准错误响应

```json
{
  "status": "error",
  "error": {
    "code": "NOT_FOUND",
    "message": "文档不存在",
    "details": {
      "slug": "nonexistent-doc"
    }
  }
}
```

### 错误代码

| 错误代码 | HTTP状态码 | 说明 |
|----------|------------|------|
| NOT_FOUND | 404 | 资源不存在 |
| INVALID_REQUEST | 400 | 请求参数无效 |
| INTERNAL_ERROR | 500 | 服务器内部错误 |

## 📝 使用示例

### JavaScript 客户端

```javascript
// 获取文档列表
async function getDocs() {
  const response = await fetch('/api/docs');
  const data = await response.json();
  return data.data;
}

// 搜索文档
async function searchDocs(query) {
  const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
  const data = await response.json();
  return data.data;
}
```

## 🔗 相关链接

- [静态网站服务文档](/docs/www) - 功能介绍
- [配置文档](/docs/config) - 配置说明
- [部署指南](/docs/deploy) - 生产环境部署

---

## NRPC API (规划中)

未来将基于 NRPC 提供更强大的 API 功能：

- **统一协议**: 基于 NRPC 的请求队列转发，统一协议
- **认证与授权**: 令牌与策略结合，确保请求边界
- **错误码与重试**: 标准化返回，便于客户端处理