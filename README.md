# MetaBase

Next Generation Backend Server designed for one-person companies and small teams.

**Goal**: Eliminate 90% of repetitive backend work, focus on 10% of important features.
**目标**: 为一人公司和小团队设计，高性能，满足后期成长需求，无复杂运维。

## 🚀 快速开始

### 当前状态
**⚠️ 早期开发阶段** - 架构设计完成，核心功能开发中

### 环境要求
- Go 1.25+
- Node.js 18+ (仅管理界面开发)

### 安装和运行
```bash
# 克隆仓库
git clone https://github.com/metabase/metabase.git
cd metabase

# 安装 Go 依赖
go mod tidy

# 构建二进制文件
go build -o bin/metabase ./cmd/metabase

# 启动开发服务器
./bin/metabase server --dev
```

### 管理界面开发
```bash
# 安装前端依赖 (在另一个终端)
cd admin-svelte
npm install
npm run dev
```

## 🌐 访问地址

- **主服务**: http://localhost:7609
- **管理控制台**: http://localhost:7609/admin (基础界面)
- **API 健康检查**: http://localhost:7609/api/health
- **静态网站**: http://localhost:8080 (基础服务器)

## 📊 开发进度

### ✅ 已完成 (基础实现)
- **项目架构**: 模块化设计，清晰的目录结构
- **CLI 工具**: 基本的命令行界面和服务器启动
- **HTTP API**: 基础的 REST API 框架
- **静态网站服务器**: 基本的文件服务功能
- **管理界面**: Svelte + Tailwind 基础框架
- **客户端库**: TypeScript 和 Python 基础结构

### 🔧 开发中 (部分完成)
- **存储引擎**: SQLite 基础集成 (Pebble KV 待开发)
- **认证系统**: API Key 管理基础框架
- **CMS 功能**: 内容管理系统基础结构

### ❌ 待开发 (核心功能)
- **嵌入式 NATS**: 消息队列和 RPC 系统
- **混合存储引擎**: SQLite + Pebble 完整实现
- **实时分析**: 访问日志、搜索、仪表盘
- **多租户系统**: 租户隔离和权限管理
- **行级安全 (RLS)**: 数据访问控制
- **文件管理系统**: 多后端文件存储
- **测试套件**: 单元测试、集成测试、E2E 测试
- **构建系统**: 完整的 Makefile 和 CI/CD

## 🏗️ 架构设计

MetaBase 采用一体化嵌入式架构设计：

```
┌─────────────────────────────────────────────────────┐
│                  MetaBase Server                    │
├─────────────────────────────────────────────────────┤
│  HTTP API  │  Embedded NATS  │  Analytics  │  Admin  │
│  (基础)     │     (待开发)    │   (待开发)   │ (基础)  │
├─────────────────────────────────────────────────────┤
│   Storage   │   Files   │   Auth   │  Tenant  │ RLS  │
│  (部分)     │  (待开发)  │  (基础)   │ (待开发) │(待开发)│
├─────────────────────────────────────────────────────┤
│   SQLite    │  Pebble  │  File System              │
│  (基础)     │ (待开发)  │   (待开发)                │
└─────────────────────────────────────────────────────┘
```

### 核心设计原则
- **零配置**: 所有服务内嵌，无外部依赖
- **高性能**: 混合存储，微秒级响应 (目标)
- **单二进制**: 简化部署和运维
- **可扩展**: 支持水平和垂直扩展

## 📖 项目结构

```
metabase/
├── cmd/metabase/          # CLI 应用入口 (✅ 基础)
├── internal/              # 内部包 (⚠️ 部分实现)
│   ├── cli/              # CLI 命令 (✅ 基础)
│   ├── server/           # 核心服务器 (⚠️ 不完整)
│   ├── www/              # 静态 web 服务 (⚠️ 基础)
│   ├── api/              # API 处理器 (✅ 基础)
│   ├── cms/              # CMS 功能 (⚠️ 基础)
│   └── keys/             # API Key 管理 (✅ 基础)
├── clients/               # 多语言客户端库
│   ├── typescript/       # TypeScript SDK (⚠️ 基础结构)
│   └── python/           # Python SDK (⚠️ 基础结构)
├── admin-svelte/          # 管理界面 (✅ 基础框架)
├── web/                   # 静态资源和 CMS (⚠️ 基础)
└── spec/                  # 规格文档 (✅ 已添加)
```

**图例**: ✅ 完成 | ⚠️ 部分/基础 | ❌ 待开发

## 🚀 开发指南

### 基本开发流程
```bash
# 1. 安装依赖
go mod tidy
cd admin-svelte && npm install

# 2. 开发模式 (后端)
go run ./cmd/metabase server --dev

# 3. 开发模式 (前端，另开终端)
cd admin-svelte && npm run dev

# 4. 运行测试 (当前覆盖率较低)
go test ./...
```

### 代码规范
- 遵循 Go 标准编码规范
- 使用 `gofmt` 格式化代码
- 前端使用 TypeScript 和 Svelte 最佳实践
- 提交信息使用约定式提交格式

## 🎯 开发路线图

对齐Supabase、Pocketbase、Loki、Grafana

## TODO
- [ ] Standalone server or embeded server.
- [ ] Setup admin and config center.
- [ ] Docker ENVORIMENT config support.
- [ ] 多租户、多项目支持。
- [ ] 静态网站Serve, 部署方便（ftp？？）
- [x] Markdown网站模板，博客新闻文档易用。
- [ ] Supabase like client API.
- [x] Sqlit + NATS + Pebble + (Redis?) = 高可用的存储引擎
- [ ] NRPC 高性能队列base on
NATS，分布式任务及RPC框架，主力，稳定性保证，削峰填谷。
- [ ] HTTP(NPRC) API Framework（Base on NRPC, Request queue to NATS)
- [ ] 存、算、管、查、搜、事务在业务模型上全部轻松支持，保持一致。
- [ ] Supabase like 行级安全策略RLS，BaaS，无后端设计. anno_key.
- [ ] 测试、生产、开发 多环境同步、部署问题。
- [ ] MQTT/MAIL/CMS/FILE/IM/AUTH/RAG/CRAWLER/RECOMMENDATION
- [-] 管理后台的界面
  - [ ] 管理员
  - [ ] 请求日志，搜索
  - [ ] 访问请求分析
  - [ ] 性能监控
  - [ ] 配置中心
  - [ ] 表管理

### 第一阶段 - 核心基础 (v0.1)
**目标**: 完成核心基础设施
- [ ] **存储引擎完整实现**
  - [ ] Pebble KV 集成
  - [ ] SQLite schema 和迁移
  - [ ] 数据访问层和事务管理
- [ ] **安全基础**
  - [ ] 输入验证和 SQL 注入防护
  - [ ] 速率限制和请求节流
  - [ ] 安全 API Key 存储
- [ ] **核心服务器**
  - [ ] NRPC 服务器和 NATS 集成
  - [ ] 请求路由和中间件
  - [ ] 错误处理和日志系统

### 第二阶段 - 功能完善 (v0.2)
**目标**: 完成核心功能，达到可用状态
- [ ] **认证和授权系统**
  - [ ] JWT 认证实现
  - [ ] RBAC 权限系统
  - [ ] 多租户基础架构
- [ ] **管理界面完善**
  - [ ] 完整的管理功能
  - [ ] 实时日志查看
  - [ ] 基础监控面板
- [ ] **测试和构建**
  - [ ] 单元测试 (目标 80% 覆盖率)
  - [ ] 集成测试
  - [ ] 完整的 Makefile

### 第三阶段 - 生产就绪 (v1.0)
**目标**: 生产环境可用
- [ ] **高级功能**
  - [ ] 实时分析和报表
  - [ ] 文件管理系统
  - [ ] 行级安全 (RLS)
  - [ ] 高级查询和搜索
- [ ] **运维工具**
  - [ ] 监控和告警
  - [ ] 备份和恢复
  - [ ] 性能优化
  - [ ] Docker 支持

### 第四阶段 - 企业功能 (v1.5+)
**目标**: 企业级功能
- [ ] **高级特性**
  - [ ] 分布式部署
  - [ ] 插件系统
  - [ ] 工作流引擎
  - [ ] API 网关功能

## 🧪 测试

### 当前测试状态
**⚠️ 测试覆盖率 < 10%** - 急需改进

```bash
# 运行现有测试
go test ./...

# 生成覆盖率报告
go test -cover ./...

# 注意: 完整的测试套件在开发中
```

### 测试计划
- **单元测试**: 每个包 >80% 覆盖率
- **集成测试**: API 端点和数据库操作
- **E2E 测试**: 关键用户流程
- **性能测试**: 负载和压力测试
- **安全测试**: 漏洞扫描和渗透测试

## 📦 性能目标

### 当前状态
- **启动时间**: ~5 秒 (目标: <2 秒)
- **内存占用**: ~50MB (目标: <100MB 空闲)
- **并发处理**: 基础测试 (目标: 10,000+ QPS)
- **响应时间**: P95 ~100ms (目标: <50ms)

### 性能优化计划
1. **存储层优化**: 查询优化、索引策略
2. **缓存系统**: 多层缓存架构
3. **连接池**: 数据库和 HTTP 连接优化
4. **并发处理**: Goroutine 池和负载均衡

## 🤝 贡献指南

### 开发流程
1. **Fork** 项目到个人仓库
2. **创建** 功能分支 (`git checkout -b feature/amazing-feature`)
3. **开发** 功能并添加测试
4. **测试** 确保所有测试通过
5. **提交** 使用约定式提交格式
6. **推送** 到个人仓库
7. **创建** Pull Request

### 约定式提交
```
feat: 添加新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 代码重构
test: 测试相关
chore: 构建工具、依赖更新
```

### 开发环境设置
```bash
# 安装开发工具
go install github.com/cosmtrek/air@latest  # 热重载
go install golang.org/x/tools/cmd/goimports@latest

# 设置 git hooks (可选)
cp scripts/pre-commit .git/hooks/
chmod +x .git/hooks/pre-commit
```

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [NATS](https://nats.io/) - 高性能消息系统
- [SQLite](https://sqlite.org/) - 可靠的嵌入式数据库
- [Pebble](https://github.com/cockroachdb/pebble) - 高性能 KV 存储
- [Svelte](https://svelte.dev/) - 现代前端框架

## 📞 联系方式

- **项目主页**: https://github.com/metabase/metabase
- **问题反馈**: https://github.com/metabase/metabase/issues
- **讨论社区**: https://github.com/metabase/metabase/discussions

---

**🚀 减轻后端开发负担，专注核心业务价值！**

**当前状态**: 积极开发中，欢迎贡献代码和反馈建议！