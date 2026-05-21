# 构建阶段使用 node 镜像编译前端产物
FROM node:18-alpine AS builder

WORKDIR /app

# 设置 npm 镜像源并安装 pnpm
RUN npm config set registry https://registry.npmmirror.com && \
    npm install -g pnpm

# 预先拷贝包管理配置并安装依赖，利用 Docker 构建缓存
COPY package.json pnpm-lock.yaml ./
RUN pnpm install

# 拷贝项目源代码并执行构建
COPY . .
RUN pnpm run build

# 运行阶段使用 nginx 提供静态资源服务
FROM nginx:alpine

# 根据 docker/conf/nginx/conf.d/geekai.conf 的配置，
# Nginx 需要在 /var/www/app/dist 读取静态页面文件
WORKDIR /var/www/app/dist

# 将构建好的前端产物拷贝到 Nginx 指定的目录
COPY --from=builder /app/dist /var/www/app/dist

# 声明前端 Nginx 暴露的端口
EXPOSE 8080

# 以前台模式启动 nginx
CMD ["nginx", "-g", "daemon off;"]
