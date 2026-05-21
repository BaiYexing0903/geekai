# 宝塔 Docker 部署指南

## 第 1 步：服务器安装 Docker

宝塔面板 → 软件商店 → 搜索 **Docker管理器** → 安装

等待安装完成（刷新页面确认状态变为"已安装"）。

---

## 第 2 步：把代码拉到服务器

在宝塔终端中执行：

```bash
cd /opt
git clone https://github.com/BaiYexing0903/geekai.git
cd geekai
```

后续更新代码只需 `git pull` 然后重新构建镜像即可。

---

## 第 3 步：准备目录结构

```bash
cd /opt/geekai/docker

# 创建数据、日志、静态资源目录
mkdir -p data/mysql/data data/mysql/init.d data/redis data/leveldb
mkdir -p logs/mysql logs/app logs/nginx
mkdir -p static/upload

# 把 SQL 初始化文件放到 MySQL 自动导入目录
cp ../database/geekai_plus-v4.2.6.sql data/mysql/init.d/
```

---

## 第 4 步：构建镜像

```bash
cd /opt/geekai

# 构建后端镜像（约 3-5 分钟，首次需下载 Go 依赖）
docker build -t geekai-api:local -f api/Dockerfile api/

# 构建前端镜像（约 3-5 分钟，首次需下载 npm 依赖）
docker build -t geekai-web:local -f web/Dockerfile web/
```

构建完成后确认镜像存在：

```bash
docker images | grep geekai
```

---

## 第 5 步：修改配置文件

### 5.1 创建 config.toml

> `config.toml` 被 `.gitignore` 排除，不会提交到仓库，需要手动创建。

```bash
cd /opt/geekai/docker/conf
cp ../../api/config.sample.toml config.toml
```

### 5.2 修改 `docker/conf/config.toml`

从 `config.sample.toml` 复制后，**必须修改以下几处**：

| 配置项 | 示例值（需修改） | 改为 |
|--------|-----------------|------|
| `MysqlDns` 中的数据库地址 | `172.22.11.200:3307` | `geekai-mysql:3306` |
| `MysqlDns` 中的数据库名 | `chatgpt_plus` | `geekai_plus` |
| `MysqlDns` 中的密码 | `12345678` | 你自己的密码 |
| `[Session] SecretKey` | `azyehq3iv...` | 随机字符串 |
| `[Redis] Host` | `localhost` | `geekai-redis` |
| `[Redis] Password` | 空 | 你自己的密码 |
| `TikaHost` | `http://tika:9998` | `http://geekai-tika:9998` |
| `[OSS.Local] BaseURL` | `http://localhost:5678/...` | `https://你的域名/static/upload` |

修改后的关键配置示例：

```toml
Listen = "0.0.0.0:5678"
ProxyURL = ""
#                                                    容器名↑ 端口用内部3306↑  数据库名↑
MysqlDns = "root:你的密码@tcp(geekai-mysql:3306)/geekai_plus?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"
StaticDir = "./static"
StaticUrl = "/static"
TikaHost = "http://geekai-tika:9998"

[Session]
  SecretKey = "f4d6dd4634c329985f903c1ddf63292c37b21efe6dbabae7eccf452f36c29289"
  MaxAge = 86400

[Redis]
  Host = "geekai-redis"
  Port = 6379
  Password = "你的密码"
  DB = 0

[OSS]
   Active = "local"
   [OSS.Local]
     BasePath = "./static/upload"
     BaseURL = "https://video.cheapertakens.com/static/upload"
```

### 5.3 修改 `docker/docker-compose.yaml` 中的密码

把里面 2 处 `weidadegpS123*` 换成你自己的密码（与 config.toml 一致）：

```yaml
geekai-mysql:
  environment:
    - MYSQL_ROOT_PASSWORD=你的密码    # ← 改这里

geekai-redis:
  command: redis-server --requirepass 你的密码  # ← 改这里
```

> 因为用的是同一个 docker-compose.yaml，所有容器在同一个网络，所以 config.toml 里继续用容器名（geekai-mysql、geekai-redis、geekai-api）通信就行，不需要改成 IP。

---

## 第 6 步：启动服务

```bash
cd /opt/geekai/docker
docker compose up -d
```

查看启动状态：

```bash
docker compose ps
```

正常应该看到 5 个容器全部 Up：

```
geekai-mysql    running    0.0.0.0:3307->3306/tcp
geekai-redis    running    0.0.0.0:6380->6379/tcp
geekai-tika     running    0.0.0.0:9999->9998/tcp
geekai-api      running    0.0.0.0:5678->5678/tcp
geekai-web      running    0.0.0.0:8080->8080/tcp
```

> 首次启动 MySQL 需要初始化数据，后端可能先报连接失败。等 30 秒后重启后端即可：
>
> ```bash
> docker compose restart geekai-api
> ```

查看后端日志确认启动成功，应该看到 `[Fx] RUNNING`：

```bash
docker compose logs geekai-api --tail 10
```

---

## 第 7 步：放行端口

宝塔面板 → 安全 → 添加端口规则 → 放行 **8080**

同时检查云服务器安全组也放行了 8080。

---

## 第 8 步：验证访问

| 地址                              | 说明                           |
| --------------------------------- | ------------------------------ |
| `http://你的IP:8080/chat`         | 前端聊天页面                   |
| `http://你的IP:8080/admin`        | 后台管理                       |
| `http://你的IP:8080/mobile`       | 移动端（手机访问自动跳转）     |

默认账号：

- 后台管理：`admin` / `admin123`
- 前端体验：`18888888888` / `12345678`

登录后台后，第一件事去 **系统设置 → AI 模型** 添加你的 API Key，否则无法使用。

---

## 第 9 步：绑定域名

1. 域名 DNS 添加 A 记录，指向服务器 IP `47.86.91.112`
2. 宝塔 → 网站 → 添加站点 → 填入域名 `video.cheapertakens.com`
3. 站点设置 → 反向代理 → 添加：
   - 代理名称：`geekai`
   - 目标 URL：`http://127.0.0.1:8080`
4. 站点设置 → SSL → 申请 Let's Encrypt 免费证书 → 开启强制 HTTPS

绑定域名后记得更新 config.toml 中的 OSS 地址：

```toml
[OSS.Local]
  BaseURL = "https://video.cheapertakens.com/static/upload"
```

然后 `docker compose restart geekai-api`。

---

## 日常运维命令

```bash
cd /opt/geekai/docker

# 查看状态
docker compose ps

# 看后端日志
docker compose logs -f geekai-api

# 只重启后端（改了配置后）
docker compose restart geekai-api

# 更新代码后重新部署
cd /opt/geekai && git pull
docker build -t geekai-api:local -f api/Dockerfile api/
docker build -t geekai-web:local -f web/Dockerfile web/
cd docker && docker compose up -d
```

---

## 常见问题

### 后端启动报错 `dial tcp 172.22.11.200:3307: connection timed out`

config.toml 中的 MySQL 地址没有改成容器名。确保 MysqlDns 中使用的是 `geekai-mysql:3306`，不是示例中的 IP 地址。

### 后端启动报错 `Unknown database 'chatgpt_plus'`

config.toml 中的数据库名错误。`config.sample.toml` 中是 `chatgpt_plus`，需要改成 `geekai_plus`。

### 前端 502 错误

后端没有正常启动，查看日志排查：

```bash
docker compose logs geekai-api 2>&1 | grep -iE "error|panic|fail" | tail -10
```

---

## 关于进阶拆分部署

官方进阶文档建议把 MySQL、Redis、Tika 拆成独立容器单独部署，好处是：

- 重启 API/Web 时不影响数据库
- 多个项目可以共用同一个 MySQL/Redis
- 更新更灵活

目前服务器如果只跑这一个项目，用上面的统一 docker-compose 部署就够了，后续有需要再拆分。

拆分时只需要：

1. 把 `docker-compose.yaml` 拆成多个文件（每个服务一份）
2. 把 config.toml 里的容器名改成宿主机 IP + 端口，例如：

```toml
# 拆分前（容器名通信）
MysqlDns = "root:密码@tcp(geekai-mysql:3306)/geekai_plus?..."

# 拆分后（IP + 端口通信）
MysqlDns = "root:密码@tcp(192.168.1.100:3307)/geekai_plus?..."

[Redis]
Host = "192.168.1.100"
Port = 6380
```
