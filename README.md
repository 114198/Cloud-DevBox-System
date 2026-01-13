# Cloud DevBox

基于 Web 界面的云原生开发环境配置和管理系统。

## 项目结构

```
cloud-devbox/
├── apps/                    # 应用程序
│   ├── web/                 # React Web 前端
│   ├── desktop/             # Tauri 桌面客户端
│   └── cli/                 # Rust CLI 工具
├── services/                # 后端服务
│   ├── gateway/             # Rust API Gateway
│   ├── core/                # Go 核心业务服务
│   ├── realtime/            # Go 实时服务
│   ├── container/           # Go 容器管理
│   └── media/               # Rust 媒体服务
├── packages/                # 共享包
│   └── shared/              # 共享类型和工具
└── database/                # 数据库迁移和 Schema
```

## 技术栈

### 后端服务
| 服务 | 语言 | 框架 |
|------|------|------|
| API Gateway | Rust | Axum |
| 核心业务服务 | Go | Gin + GORM |
| 实时协作服务 | Go | gorilla/websocket |
| 容器管理服务 | Go | client-go |
| 媒体服务 | Rust | mediasoup-rs |

### 前端应用
| 平台 | 技术 | 框架 |
|------|------|------|
| Web 端 | TypeScript | React 18 + Vite |
| 桌面客户端 | Rust | Tauri 2.0 |
| CLI 工具 | Rust | clap |

### 数据层
- PostgreSQL 16: 主数据库
- Redis 7: 缓存、会话、实时数据
- NATS: 高性能消息队列

## 开发指南

### 前置要求
- Node.js 18+
- Rust 1.70+
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7

### 快速开始

```bash
# 克隆项目
git clone https://github.com/your-org/cloud-devbox.git
cd cloud-devbox

# 安装前端依赖
cd apps/web && npm install

# 构建 Rust 服务
cargo build --workspace

# 构建 Go 服务
cd services/core && go build ./...
```

## 许可证

MIT License
