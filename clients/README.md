# MetaBase Client Libraries

多语言客户端SDK，提供对MetaBase核心功能的完整访问。

## 🚀 快速开始

选择你偏好的编程语言：

- **[Go](./go/)** - 完整实现，支持所有核心功能
- **[TypeScript](./typescript/)** - 开发中 🚧
- **[Python](./python/)** - 开发中 🚧

## ✨ 功能特性

### 🟢 已实现 (Go)
- ✅ HTTP客户端封装
- ✅ 认证管理 (API Key, JWT)
- ✅ CRUD操作接口
- ✅ 文件上传/下载
- ✅ 实时订阅
- ✅ 错误处理和重试
- ✅ 会话管理
- ✅ 多租户支持

### 🟡 开发中
- 文件管理接口
- 行级安全策略支持
- 实时数据同步
- 离线缓存
- 批量操作

### 📋 计划中
- 本地数据缓存
- 响应式流
- 插件系统
- GraphQL支持

## 💡 使用示例

### Go
```go
import "github.com/metabase/metabase/clients/go"

config := &client.Config{
    URL:     "http://localhost:7609",
    APIKey:  "your-api-key",
}

client := client.New(config)
result, err := client.Query(ctx, "users", &QueryOptions{
    Limit: 10,
})
```

### TypeScript (开发中)
```typescript
import { MetaBaseClient } from '@metabase/clients';

const client = new MetaBaseClient({
  url: 'http://localhost:7609',
  apiKey: 'your-api-key'
});

const users = await client.query('users', { limit: 10 });
```

### Python (开发中)
```python
from metabase_clients import MetaBaseClient

client = MetaBaseClient(
    url='http://localhost:7609',
    api_key='your-api-key'
)

users = client.query('users', limit=10)
```

## 🔗 相关链接

- [MetaBase 主项目](../README.md)
- [API 文档](../docs/api.md)
- [开发指南](../docs/start.md)
- [贡献指南](../CONTRIBUTING.md)

## 📄 许可证

本项目遵循 [MIT 许可证](../LICENSE)。