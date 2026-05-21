FROM golang:1.22-alpine AS builder

WORKDIR /build

# 设置 Go 环境变量以加速国内下载和配置构建参数
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 预先下载依赖以利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 拷贝代码并构建可执行文件
COPY . .
RUN go build -o geekai-api main.go

# 运行阶段使用轻量级 alpine 镜像
FROM alpine:latest

WORKDIR /var/www/app

# 从构建阶段拷贝编译好的二进制文件
COPY --from=builder /build/geekai-api /var/www/app/

# 安装时区和证书（部分 HTTP 请求和时间相关功能需要）
RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 声明应用暴露的端口
EXPOSE 5678

# 启动命令
ENTRYPOINT ["./geekai-api"]
