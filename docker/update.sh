#!/bin/bash
# geekai 一键更新部署脚本
# 用法: cd /opt/geekai/docker && bash update.sh

set -e

cd /opt/geekai

echo "==> [1/4] 拉取最新代码..."
git pull

echo "==> [2/4] 构建后端镜像..."
docker build -t geekai-api:local -f api/Dockerfile api/

echo "==> [3/4] 构建前端镜像..."
docker build -t geekai-web:local -f web/Dockerfile web/

echo "==> [4/4] 重启服务..."
cd docker
docker compose up -d

echo ""
echo "✓ 部署完成！查看状态: docker compose ps"
echo "查看后端日志: docker compose logs -f geekai-api"
