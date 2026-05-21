# GateHub 部署架构说明

## 目录结构

```
deploy/
├── README.md                 # 本文件 - 快速开始指南
├── deploy.md                 # 详细部署文档
├── push.env.example          # 配置文件模板
├── docker-compose.deploy.yml # 生产环境部署配置
├── docker-compose.local.yml  # 本地开发部署配置
│
├── backend/                  # 后端构建
│   └── build-and-push.sh     # 后端构建脚本
│
├── frontend/                 # 前端构建
│   ├── build-and-push.sh     # 前端构建脚本
│   ├── Dockerfile            # 前端 Dockerfile (完整构建)
│   ├── Dockerfile.simple     # 前端 Dockerfile (使用已构建的 dist)
│   ├── Dockerfile.frontend   # 前端构建产物 (仅用于特殊场景)
│   └── nginx.conf            # Nginx 配置
│
└── jenkins/                  # Jenkins 流水线
    ├── Jenkinsfile.backend   # 后端构建流水线
    └── Jenkinsfile.frontend  # 前端构建流水线
```

## 快速开始

### 本地部署（开发环境）

```bash
# 1. 构建前端 (使用 latest 标签)
cd deploy/frontend
bash build-and-push.sh --no-push

# 2. 构建后端 (使用 latest 标签)
cd ../backend
bash build-and-push.sh --no-push

# 3. 启动所有服务
POSTGRES_PASSWORD=weidada123 docker-compose -f docker-compose.local.yml up -d
```

### 生产部署 (Jenkins 自动构建)

Jenkins 会自动使用构建编号作为版本号（例如：`build-42`）：

```bash
# Jenkins 会自动执行：
# - 前端：gatehub-frontend:build-42 + :latest
# - 后端：gatehub-server:build-42 + :latest
```

### 手动推送（可选）

```bash
# 1. 复制配置文件
cp push.env.example push.env
# 编辑 push.env，填入 Harbor 账号密码

# 2. 构建并推送前端
cd deploy/frontend
bash build-and-push.sh

# 3. 构建并推送后端
cd ../backend
bash build-and-push.sh

# 4. 部署到生产环境
POSTGRES_PASSWORD=weidada123 docker-compose -f docker-compose.deploy.yml up -d
```

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                         CI/CD Flow                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────┐    ┌───────────┐    ┌─────────┐              │
│  │   Git    │───>│  Jenkins  │───>│ Harbor  │              │
│  │  GateHub│    │  Pipeline │    │  Registry              │
│  └──────────┘    └───────────┘    └─────────┘              │
│                         │                  │                │
│                         │                  v                │
│                         │         ┌──────────────┐         │
│                         │         │  Production  │         │
│                         │         │  Deployment  │         │
│                         │         └──────────────┘         │
│                         │                                   │
│                         v                                   │
│                  ┌──────────────┐                          │
│                  │ Local Deploy │                          │
│                  └──────────────┘                          │
│                                                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Production Architecture                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   User                                                      │
│    │                                                        │
│    v                                                        │
│  ┌─────────────────┐                                        │
│  │  gatehub-      │  Port 8080                            │
│  │  frontend       │  (Nginx + Static Files)               │
│  │  (Container)    │                                        │
│  └────────┬────────┘                                        │
│           │ API Proxy                                        │
│           v                                                  │
│  ┌─────────────────┐                                        │
│  │  gatehub-api   │  Port 3000 (Internal)                 │
│  │  (Container)    │  (Go Backend)                         │
│  └────────┬────────┘                                        │
│           │                                                  │
│    ┌──────┴──────┐                                          │
│    v             v                                          │
│ ┌──────┐    ┌────────┐                                     │
│ │Redis │    │Postgres│                                     │
│ │:6379 │    │:5432   │                                     │
│ └──────┘    └────────┘                                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 环境变量说明

### push.env 配置

```env
# Harbor 配置
HARBOR_HOST=64.32.12.251:28077
HARBOR_USERNAME=your_account
HARBOR_PASSWORD=your_password

# 镜像配置
HARBOR_PROJECT=gatehub
VERSION=latest

# 构建选项
BUILD_FRONTEND=false    # 是否同时构建前端
PUSH_LATEST=true        # 是否推送 latest 标签
```

## 常用命令

### 查看服务状态
```bash
docker ps --filter "name=gatehub"
```

### 查看日志
```bash
docker logs -f gatehub-api
docker logs -f gatehub-frontend
docker logs -f gatehub-postgres
```

### 重启服务
```bash
POSTGRES_PASSWORD=weidada123 docker-compose -f docker-compose.deploy.yml up -d
```

### 停止服务
```bash
docker-compose -f docker-compose.deploy.yml down
```

### 清理镜像
```bash
docker rmi $(docker images | grep gatehub | awk '{print $3}')
```

## Jenkins 配置

### 前置要求

1. 安装插件
   - Docker Pipeline
   - NodeJS
   - Credentials Binding

2. 配置凭据
   - `harbor-credentials`: Harbor 登录账号
   - `gitlab-credentials`: GitLab SSH key

3. 配置工具
   - Docker: docker
   - NodeJS: nodejs-18

### 创建 Job

1. 新建 Multibranch Pipeline
2. Source: GitLab
3. Script Path: `deploy/jenkins/Jenkinsfile.backend` 或 `Jenkinsfile.frontend`

## 故障排查

### 前端无法访问
```bash
# 检查容器状态
docker ps | grep gatehub-frontend

# 检查端口占用
netstat -ano | findstr :8080

# 查看容器日志
docker logs gatehub-frontend
```

### 后端 API 无法连接
```bash
# 检查后端状态
docker ps | grep gatehub-api

# 测试 API
curl http://localhost:8080/api/status

# 查看后端日志
docker logs gatehub-api
```

### 数据库连接失败
```bash
# 检查 PostgreSQL 状态
docker ps | grep gatehub-postgres

# 测试数据库连接
docker exec -it gatehub-postgres psql -U root -d gatehub
```

### 本地测试

#### 镜像打包
**前端**
```
sh deploy\frontend\build-and-push.sh --no-push
```
**后端**
```
sh deploy\backend\build-and-push.sh --no-push
```

#### 服务启动
**注意**
- pql的密码是否设置正确
- 端口和其他项目是否冲突
```
docker compose -p werouter -f deploy\docker-compose.deploy.yml up -d
```
