# GateHub 容器化部署快速参考

## 快速开始

### 1. 配置 Harbor 连接

```bash
# 复制配置文件模板
cp deploy/push.env.example deploy/push.env

# 编辑配置文件
# 修改 HARBOR_HOST, HARBOR_USERNAME, HARBOR_PASSWORD, VERSION 等
```

### 2. 登录 Harbor (可选，如果配置了账号密码则自动登录)

```bash
docker login ${HARBOR_HOST}
```

### 3. 构建并推送镜像

```bash
# 仅后端
./deploy/build-and-push.sh

# 后端 + 前端
BUILD_FRONTEND=true ./deploy/build-and-push.sh
```

### 4. 部署服务

```bash
# 使用 Docker Compose 部署
docker-compose -f deploy/docker-compose.deploy.yml up -d

# 查看服务状态
docker-compose -f deploy/docker-compose.deploy.yml ps

# 查看日志
docker-compose -f deploy/docker-compose.deploy.yml logs -f gatehub

# 停止服务
docker-compose -f deploy/docker-compose.deploy.yml down
```

---

## 目录结构

```
deploy/
├── README.md                      # 详细部署文档
├── build-and-push.sh              # 构建和推送脚本
├── docker-compose.deploy.yml      # 部署配置
├── QUICK_REFERENCE.md             # 本文档
└── frontend/
    ├── Dockerfile                 # 前端 Dockerfile
    └── nginx.conf                 # Nginx 配置
```

---

## 常用命令

### 镜像管理

```bash
# 查看本地镜像
docker images | grep gatehub-server

# 删除镜像
docker rmi ${HARBOR_HOST}/${HARBOR_PROJECT}/gatehub-server:${VERSION}

# 拉取镜像
docker pull ${HARBOR_HOST}/${HARBOR_PROJECT}/gatehub-server:${VERSION}
```

### 容器管理

```bash
# 查看运行中的容器
docker ps | grep gatehub

# 查看容器日志
docker logs -f gatehub-api

# 进入容器
docker exec -it gatehub-api /bin/sh

# 重启服务
docker-compose -f deploy/docker-compose.deploy.yml restart gatehub
```

### 数据库管理

```bash
# 进入 PostgreSQL 容器
docker exec -it gatehub-postgres psql -U root -d gatehub

# 数据库备份
docker exec gatehub-postgres pg_dump -U root gatehub > backup.sql

# 数据库恢复
cat backup.sql | docker exec -i gatehub-postgres psql -U root -d gatehub
```

---

## 配置选项

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| HARBOR_HOST | Harbor 仓库地址 | harbor.example.com |
| HARBOR_PROJECT | Harbor 项目名称 | gatehub |
| VERSION | 镜像版本 | latest |
| POSTGRES_PASSWORD | PostgreSQL 密码 | changeme123! |
| BUILD_FRONTEND | 是否构建前端镜像 | false |

### 端口映射

| 服务 | 端口 | 说明 |
|------|------|------|
| gatehub-server | 3000:3000 | API 服务 |
| gatehub-frontend | 80:80 | 前端服务 (可选) |
| postgres | - | 仅容器间访问 |
| redis | - | 仅容器间访问 |

---

## 生产环境建议

### 1. 安全配置

```bash
# 修改默认密码
export POSTGRES_PASSWORD=$(openssl rand -base64 32)

# 设置 SESSION_SECRET
echo "SESSION_SECRET=$(openssl rand -hex 32)" >> .env
```

### 2. 资源限制

在 `docker-compose.deploy.yml` 中添加：

```yaml
services:
  gatehub-server:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### 3. 日志轮转

创建 `docker-compose.override.yml`:

```yaml
services:
  gatehub-server:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
```

---

## 故障排查

### 服务启动失败

```bash
# 查看详细日志
docker-compose -f deploy/docker-compose.deploy.yml logs gatehub-server

# 检查依赖服务
docker-compose -f deploy/docker-compose.deploy.yml ps

# 重新创建容器
docker-compose -f deploy/docker-compose.deploy.yml up -d --force-recreate
```

### 数据库连接失败

```bash
# 测试数据库连接
docker exec gatehub-api wget -q -O - http://postgres:5432

# 检查 PostgreSQL 状态
docker-compose -f deploy/docker-compose.deploy.yml logs postgres
```

### 镜像拉取失败

```bash
# 检查 Harbor 登录
docker logout ${HARBOR_HOST}
docker login ${HARBOR_HOST}

# 检查网络
curl -I https://${HARBOR_HOST}
```

---

## 版本升级流程

```bash
# 1. 停止当前服务
docker-compose -f deploy/docker-compose.deploy.yml down

# 2. 备份数据
docker exec gatehub-postgres pg_dump -U root gatehub > backup_$(date +%Y%m%d).sql

# 3. 拉取新镜像
export VERSION="v1.1.0"
docker pull ${HARBOR_HOST}/${HARBOR_PROJECT}/gatehub-server:${VERSION}

# 4. 启动新版本
docker-compose -f deploy/docker-compose.deploy.yml up -d

# 5. 验证服务
curl http://localhost:3000/api/status
```

---

## 支持

- 项目文档：`/deploy/README.md`
- GitHub Issues: https://github.com/QuantumNous/gatehub/issues
