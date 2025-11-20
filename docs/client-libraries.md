---
title: Client Libraries - 多语言客户端库
description: MetaBase提供TypeScript、Go、Python等多语言客户端库，支持Supabase-like API设计，让前后端开发更加简单高效
order: 30
section: getting-started
tags: [client, sdk, typescript, go, python, api]
category: docs
---

# Client Libraries - 多语言客户端库

MetaBase提供类型安全的多语言客户端库，采用Supabase-like API设计，让开发者能够快速集成MetaBase的强大功能到各种应用中。

## 🚀 支持的语言

### TypeScript / JavaScript
- **前端框架**: React, Vue, Angular, Svelte
- **运行环境**: Node.js, Deno, Browser
- **特性**: 类型安全、Promise/async-await、Tree-shaking

### Go
- **应用类型**: Web服务、微服务、CLI工具
- **特性**: 强类型、高性能、并发安全
- **集成**: 标准库、Gin、Echo等框架

### Python
- **应用类型**: Web应用、数据科学、自动化脚本
- **特性**: 简洁语法、丰富的生态系统、异步支持
- **框架**: Django, Flask, FastAPI集成

## 📦 安装和设置

### TypeScript / JavaScript

```bash
# npm
npm install @metabase/client

# yarn
yarn add @metabase/client

# pnpm
pnpm add @metabase/client
```

```typescript
import { createClient } from '@metabase/client';

// 创建客户端
const metabase = createClient({
  url: 'https://your-metabase-instance.com',
  apikey: 'your-api-key',
});
```

### Go

```bash
go get github.com/metabase/metabase/internal/clientlib/go
```

```go
import (
    "github.com/metabase/metabase/internal/clientlib/go"
)

// 创建客户端
config := &clientlib.Config{
    URL:    "https://your-metabase-instance.com",
    APIKey: "your-api-key",
}

client := clientlib.NewClient(config)
```

### Python

```bash
pip install metabase-client
```

```python
from metabase_client import MetaBaseClient, ClientConfig

# 创建客户端
config = ClientConfig(
    url="https://your-metabase-instance.com",
    apikey="your-api-key"
)

client = MetaBaseClient(config)
```

## 🔐 认证管理

### 用户注册和登录

#### TypeScript
```typescript
// 用户注册
const { data, error } = await metabase.auth.signUp({
  email: 'user@example.com',
  password: 'password123',
  options: {
    data: {
      display_name: 'John Doe'
    }
  }
});

// 用户登录
const { data, error } = await metabase.auth.signIn({
  email: 'user@example.com',
  password: 'password123'
});

// 获取当前用户
const { data: user } = await metabase.auth.getUser();

// 登出
const { error } = await metabase.auth.signOut();
```

#### Go
```go
// 用户注册
authResp, err := client.Auth().SignUp(ctx, "user@example.com", "password123", map[string]interface{}{
    "display_name": "John Doe",
})

// 用户登录
authResp, err := client.Auth().SignIn(ctx, "user@example.com", "password123")

// 获取当前用户
user, err := client.Auth().GetUser(ctx)

// 登出
err = client.Auth().SignOut(ctx)
```

#### Python
```python
# 用户注册
auth_response = client.auth.sign_up(
    email="user@example.com",
    password="password123",
    options={"display_name": "John Doe"}
)

# 用户登录
auth_response = client.auth.sign_in(
    email="user@example.com",
    password="password123"
)

# 获取当前用户
user_response = client.auth.get_user()

# 登出
response = client.auth.sign_out()
```

### Session管理

#### TypeScript
```typescript
// 设置访问令牌
await metabase.auth.setSession('your-jwt-token');

// 自动刷新令牌
const client = createClient({
  url: 'https://your-metabase-instance.com',
  auth: {
    autoRefreshToken: true,
    persistSession: true
  }
});
```

#### Go
```go
// 设置访问令牌
err := client.Auth().SetSession(ctx, "your-jwt-token")

// 自动刷新令牌配置
config := &clientlib.Config{
    URL: "https://your-metabase-instance.com",
    Auth: &clientlib.AuthConfig{
        AutoRefreshToken: true,
        PersistSession:   true,
    },
}
```

#### Python
```python
# 设置访问令牌
response = client.auth.set_session("your-jwt-token")

# 自动刷新令牌配置
config = ClientConfig(
    url="https://your-metabase-instance.com",
    auth=AuthConfig(
        auto_refresh_token=True,
        persist_session=True
    )
)
```

## 🗄️ 数据库操作

### 查询数据

#### TypeScript
```typescript
// 简单查询
const { data, error } = await metabase
  .from('users')
  .select('*');

// 带条件查询
const { data, error } = await metabase
  .from('users')
  .select('id, name, email')
  .eq('active', true)
  .order('created_at', { ascending: false })
  .limit(10);

// 复杂查询
const { data, error } = await metabase
  .from('posts')
  .select(`
    id,
    title,
    content,
    users (
      id,
      name,
      avatar_url
    ),
    comments (
      id,
      content,
      created_at
    )
  `)
  .in('status', ['published', 'featured'])
  .gte('created_at', '2024-01-01')
  .order('published_at', { ascending: false });

// 单条记录查询
const { data, error } = await metabase
  .from('users')
  .select('*')
  .eq('email', 'user@example.com')
  .single();
```

#### Go
```go
// 简单查询
response, err := client.From("users").Select("*").Execute()

// 带条件查询
response, err := client.From("users").
    Select("id, name, email").
    Where("active", "=", true).
    Order("created_at", false).
    Limit(10).
    Execute()

// 复杂查询
response, err := client.From("posts").
    Select("id, title, content").
    Where("status", "in", []interface{}{"published", "featured"}).
    Gte("created_at", "2024-01-01").
    Order("published_at", false).
    Execute()

// 单条记录查询
response, err := client.From("users").
    Select("*").
    Where("email", "=", "user@example.com").
    Single()
```

#### Python
```python
# 简单查询
response = client.from_("users").select("*").execute()

# 带条件查询
response = client.from_("users").select("id, name, email")\
    .where("active", "=", True)\
    .order("created_at", ascending=False)\
    .limit(10)\
    .execute()

# 复杂查询
response = client.from_("posts").select("id, title, content")\
    .where("status", "in", ["published", "featured"])\
    .gte("created_at", "2024-01-01")\
    .order("published_at", ascending=False)\
    .execute()

# 单条记录查询
response = client.from_("users").select("*")\
    .where("email", "=", "user@example.com")\
    .single()
```

### 插入数据

#### TypeScript
```typescript
// 插入单条记录
const { data, error } = await metabase
  .from('users')
  .insert({
    name: 'John Doe',
    email: 'john@example.com',
    active: true
  })
  .select();

// 批量插入
const { data, error } = await metabase
  .from('users')
  .insert([
    { name: 'Alice', email: 'alice@example.com' },
    { name: 'Bob', email: 'bob@example.com' },
    { name: 'Charlie', email: 'charlie@example.com' }
  ])
  .select();
```

#### Go
```go
// 插入单条记录
userData := map[string]interface{}{
    "name":   "John Doe",
    "email":  "john@example.com",
    "active": true,
}
response, err := client.Post(ctx, "/data/users", userData)

// 批量插入
users := []map[string]interface{}{
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob", "email": "bob@example.com"},
    {"name": "Charlie", "email": "charlie@example.com"},
}
response, err := client.Post(ctx, "/data/users", map[string]interface{}{
    "records": users,
})
```

#### Python
```python
# 插入单条记录
response = client._request("POST", "/data/users", {
    "name": "John Doe",
    "email": "john@example.com",
    "active": True
})

# 批量插入
users = [
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob", "email": "bob@example.com"},
    {"name": "Charlie", "email": "charlie@example.com"}
]
response = client._request("POST", "/data/users", {"records": users})
```

### 更新数据

#### TypeScript
```typescript
// 更新记录
const { data, error } = await metabase
  .from('users')
  .update({
    last_login: new Date().toISOString(),
    active: true
  })
  .eq('id', '123e4567-e89b-12d3-a456-426614174000')
  .select();

// 批量更新
const { data, error } = await metabase
  .from('posts')
  .update({ status: 'archived' })
  .lt('created_at', '2023-01-01')
  .eq('status', 'published');
```

#### Go
```go
// 更新记录
updateData := map[string]interface{}{
    "last_login": time.Now().Format(time.RFC3339),
    "active":     true,
}
response, err := client.Put(ctx, "/data/users/123e4567-e89b-12d3-a456-426614174000", updateData)
```

#### Python
```python
# 更新记录
update_data = {
    "last_login": datetime.now().isoformat(),
    "active": True
}
response = client._request("PUT", "/data/users/123e4567-e89b-12d3-a456-426614174000", update_data)
```

### 删除数据

#### TypeScript
```typescript
// 删除记录
const { data, error } = await metabase
  .from('users')
  .delete()
  .eq('id', '123e4567-e89b-12d3-a456-426614174000');

// 批量删除
const { data, error } = await metabase
  .from('sessions')
  .delete()
  .lt('expires_at', new Date().toISOString());
```

#### Go
```go
// 删除记录
response, err := client.Delete(ctx, "/data/users/123e4567-e89b-12d3-a456-426614174000")
```

#### Python
```python
# 删除记录
response = client._request("DELETE", "/data/users/123e4567-e89b-12d3-a456-426614174000")
```

## 📁 文件存储

### 上传文件

#### TypeScript
```typescript
// 上传文件
const fileInput = document.getElementById('file-input');
const file = fileInput.files[0];

const { data, error } = await metabase.storage
  .from('avatars')
  .upload(`public/${file.name}`, file, {
    cacheControl: '3600',
    upsert: false
  });

// 获取公共URL
const publicURL = metabase.storage
  .from('avatars')
  .getPublicUrl(`public/${file.name}`);

// 下载文件
const { data, error } = await metabase.storage
  .from('documents')
  .download('report.pdf');
```

#### Go
```go
// 上传文件
fileData, err := os.ReadFile("avatar.jpg")
if err != nil {
    log.Fatal(err)
}

options := map[string]interface{}{
    "cacheControl": "3600",
    "upsert":       false,
}

response, err := client.Storage().From("avatars").
    Upload(ctx, "public/avatar.jpg", fileData, options)

// 获取公共URL
publicURL := client.Storage().From("avatars").GetPublicUrl("public/avatar.jpg")

// 下载文件
downloadData, err := client.Storage().From("documents").
    Download(ctx, "report.pdf")
```

#### Python
```python
# 上传文件
with open("avatar.jpg", "rb") as f:
    file_data = f.read()

options = {
    "cacheControl": "3600",
    "upsert": False
}

response = client.storage.from_("avatars").upload(
    "public/avatar.jpg",
    file_data,
    options
)

# 获取公共URL
public_url = client.storage.from_("avatars").get_public_url("public/avatar.jpg")

# 下载文件
download_data = client.storage.from_("documents").download("report.pdf")
```

## 🔄 实时功能

### 订阅数据变更

#### TypeScript
```typescript
// 订听表变更
const subscription = metabase
  .channel('public:users')
  .on('postgres_changes',
    {
      event: '*',
      schema: 'public',
      table: 'users'
    },
    (payload) => {
      console.log('Change received!', payload);
      switch (payload.eventType) {
        case 'INSERT':
          console.log('New user:', payload.new);
          break;
        case 'UPDATE':
          console.log('Updated user:', payload.new);
          break;
        case 'DELETE':
          console.log('Deleted user:', payload.old);
          break;
      }
    }
  )
  .subscribe();

// 订听自定义事件
const customSubscription = metabase
  .channel('user-events')
  .on('broadcast', { event: 'user-login' }, (payload) => {
    console.log('User logged in:', payload.payload);
  })
  .subscribe();

// 取消订阅
subscription.unsubscribe();
```

#### Go
```go
// 订听数据变更
subscription, err := client.Realtime().Channel("public:users").
    On("postgres_changes", func(payload interface{}) {
        fmt.Printf("Change received: %+v\n", payload)
    }).
    Subscribe(ctx)

// 订听自定义事件
customSubscription, err := client.Realtime().Channel("user-events").
    On("broadcast", func(payload interface{}) {
        fmt.Printf("User event: %+v\n", payload)
    }).
    Subscribe(ctx)

// 取消订阅
subscription.Unsubscribe(ctx)
```

#### Python
```python
# 订听数据变更
def on_user_change(payload):
    print(f"Change received: {payload}")
    event_type = payload.get("eventType")
    if event_type == "INSERT":
        print(f"New user: {payload.get('new')}")
    elif event_type == "UPDATE":
        print(f"Updated user: {payload.get('new')}")
    elif event_type == "DELETE":
        print(f"Deleted user: {payload.get('old')}")

subscription = client.realtime.channel("public:users")\
    .on("postgres_changes", on_user_change)\
    .subscribe()

# 订听自定义事件
def on_user_login(payload):
    print(f"User logged in: {payload}")

custom_subscription = client.realtime.channel("user-events")\
    .on("broadcast", on_user_login)\
    .subscribe()

# 取消订阅
subscription.unsubscribe()
```

## 🔧 高级功能

### 事务处理

#### TypeScript
```typescript
// 事务操作（示例）
import { createClient } from '@metabase/client';

const metabase = createClient({
  url: process.env.METABASE_URL,
  apikey: process.env.METABASE_KEY,
});

async function transferFunds(fromId: string, toId: string, amount: number) {
  // 开始事务
  const { data: fromAccount } = await metabase
    .from('accounts')
    .select('balance')
    .eq('id', fromId)
    .single();

  if (fromAccount.balance < amount) {
    throw new Error('Insufficient funds');
  }

  // 执行转账
  const operations = [
    // 扣除源账户
    metabase
      .from('accounts')
      .update({ balance: fromAccount.balance - amount })
      .eq('id', fromId),

    // 增加目标账户
    metabase
      .from('accounts')
      .update({
        balance: metabase.sql`balance + ${amount}`
      })
      .eq('id', toId),

    // 记录交易
    metabase
      .from('transactions')
      .insert({
        from_account_id: fromId,
        to_account_id: toId,
        amount,
        status: 'completed'
      })
  ];

  // 并行执行操作
  const results = await Promise.all(operations);

  return results;
}
```

### 数据库函数调用

#### TypeScript
```typescript
// 调用数据库函数
const { data, error } = await metabase
  .rpc('calculate_user_stats', {
    user_id: '123e4567-e89b-12d3-a456-426614174000'
  });

// 调用存储过程
const { data, error } = await metabase
  .rpc('create_user_profile', {
    p_name: 'John Doe',
    p_email: 'john@example.com',
    p_metadata: { role: 'admin' }
  });
```

## 🎯 最佳实践

### 1. 错误处理

#### TypeScript
```typescript
import { createClient, ApiError } from '@metabase/client';

const metabase = createClient(config);

async function safeOperation() {
  try {
    const { data, error } = await metabase
      .from('users')
      .select('*')
      .eq('active', true);

    if (error) {
      // 处理API错误
      if (error.code === 'PGRST116') {
        console.log('No rows found');
      } else {
        throw error;
      }
    }

    return data;
  } catch (error) {
    if (error instanceof ApiError) {
      console.error('API Error:', error.message);
    } else {
      console.error('Unexpected error:', error);
    }
    throw error;
  }
}
```

### 2. 性能优化

#### TypeScript
```typescript
// 使用select减少数据传输
const { data } = await metabase
  .from('posts')
  .select('id, title, created_at') // 只选择需要的字段
  .eq('published', true)
  .order('created_at', { ascending: false })
  .limit(20);

// 使用分页
async function getPosts(page = 1, pageSize = 20) {
  const from = (page - 1) * pageSize;
  const to = from + pageSize - 1;

  const { data } = await metabase
    .from('posts')
    .select('*')
    .range(from, to)
    .order('created_at', { ascending: false });

  return data;
}

// 批量操作
const users = [
  { name: 'Alice', email: 'alice@example.com' },
  { name: 'Bob', email: 'bob@example.com' }
];

const { data } = await metabase
  .from('users')
  .insert(users)
  .select();
```

### 3. 类型安全

#### TypeScript
```typescript
// 定义数据库类型
interface Database {
  public: {
    Tables: {
      users: {
        Row: {
          id: string;
          name: string;
          email: string;
          active: boolean;
          created_at: string;
          updated_at: string;
        };
        Insert: {
          name: string;
          email: string;
          active?: boolean;
        };
        Update: {
          name?: string;
          email?: string;
          active?: boolean;
        };
      };
    };
  };
}

// 创建类型化客户端
const metabase = createClient<Database>(config);

// 类型安全的查询
const { data: users } = await metabase
  .from('users')
  .select('id, name, email')
  .eq('active', true);

// 类型安全的插入
const { data: newUser } = await metabase
  .from('users')
  .insert({
    name: 'John Doe',
    email: 'john@example.com',
    // TypeScript会检查必需字段
  })
  .select()
  .single();
```

### 4. 缓存策略

#### TypeScript
```typescript
import { createClient } from '@metabase/client';

const metabase = createClient({
  url: process.env.METABASE_URL,
  apikey: process.env.METABASE_KEY,
  // 配置缓存
  db: {
    schema: 'public',
    // 实现查询缓存
    fetch: async (url, options) => {
      const cacheKey = `metabase:${url}`;

      // 检查缓存
      const cached = localStorage.getItem(cacheKey);
      if (cached) {
        const { data, timestamp } = JSON.parse(cached);
        const age = Date.now() - timestamp;

        // 5分钟缓存
        if (age < 5 * 60 * 1000) {
          return { data };
        }
      }

      // 执行请求
      const response = await fetch(url, options);
      const data = await response.json();

      // 缓存结果
      localStorage.setItem(cacheKey, JSON.stringify({
        data,
        timestamp: Date.now()
      }));

      return { data };
    }
  }
});
```

## 📚 更多资源

### 示例项目
- [React + TypeScript 示例](https://github.com/metabase/examples-react)
- [Vue.js 示例](https://github.com/metabase/examples-vue)
- [Next.js 示例](https://github.com/metabase/examples-nextjs)
- [Go Web服务示例](https://github.com/metabase/examples-go)
- [Python Flask示例](https://github.com/metabase/examples-python)

### 文档链接
- [API参考文档](./api.md)
- [身份验证指南](./auth.md)
- [实时功能详解](./realtime.md)
- [文件存储指南](./storage.md)

MetaBase的客户端库提供了统一的API设计，无论使用哪种语言，都能享受一致的开发体验。通过类型安全、错误处理和性能优化等特性，开发者可以快速构建可靠的应用程序。