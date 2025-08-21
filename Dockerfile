# === Stage 1: Downloader ===
# 这个阶段专门负责下载并预置一个最新版的 CloudflareSpeedTest
FROM alpine:latest AS downloader

# 安装下载和处理所需的工具
RUN apk add --no-cache curl tar gzip jq

# 设置工作目录
WORKDIR /download

# ARG TARGETARCH 会被 Docker 自动设置为 amd64, arm64 等
ARG TARGETARCH
# 从 GitHub API 获取最新 Release 信息, 并解析出对应架构的下载链接
RUN LATEST_URL=$(curl -s "https://api.github.com/repos/XIU2/CloudflareSpeedTest/releases/latest" | \
    jq -r ".assets[] | select(.name | contains(\"linux_${TARGETARCH}\")) | .browser_download_url") && \
    echo "Downloading baked-in version from ${LATEST_URL}" && \
    curl -L -o cfst.tar.gz "${LATEST_URL}"

# 解压下载的压缩包
RUN tar -zxvf cfst.tar.gz


# === Stage 2: Go Builder ===
# 这个阶段负责编译我们自己的 cfst-client 应用
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 编译我们的客户端程序
RUN CGO_ENABLED=0 GOOS=linux go build -o test-client ./cmd/main.go


# === Stage 3: Final Image ===
# 这是最终的运行镜像
FROM alpine

# [核心修复] 安装 bash 和其他必要依赖，解决 "can't execute 'bash'" 错误
RUN apk add --no-cache ca-certificates curl tar gzip bash

# 设置工作目录
WORKDIR /app

# 创建用于挂载的 config 目录
RUN mkdir -p /app/config

# --- 预置 CloudflareSpeedTest 程序 ---
# 从 Downloader 阶段拷贝已解压的文件，作为 "内置" 版本
# 拷贝可执行文件并赋予权限
COPY --from=downloader /download/cfst /usr/local/bin/CloudflareSpeedTest
# 拷贝 IP 列表文件到 config 目录，如果用户没有挂载自己的版本，则使用内置的
COPY --from=downloader /download/ip.txt /app/config/ip.txt
COPY --from=downloader /download/ipv6.txt /app/config/ipv6.txt

# --- 部署我们自己的客户端程序 ---
# 从 Go Builder 阶段拷贝我们编译好的应用
COPY --from=builder /app/test-client .

# 设置环境变量
ENV GITHUB_TOKEN="" \
    GITHUB_PROXY=""

# 设置容器启动命令
# 程序启动后，依然会运行 installer 逻辑，如果发现有更新的版本，会覆盖掉上面内置的文件
ENTRYPOINT ["./test-client"]