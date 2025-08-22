# === Stage 1: Downloader ===
# 这个阶段专门负责下载并解压最新版的 CloudflareSpeedTest
FROM alpine:latest AS downloader

# 安装下载和处理所需的工具
RUN apk add --no-cache curl tar gzip jq

# 设置工作目录
WORKDIR /download

# ARG TARGETARCH 会被 Docker 自动设置为 amd64, arm64 等
ARG TARGETARCH
# [核心修改]
# 1. 先获取完整的 latest release JSON 数据
# 2. 从中分别提取下载链接 (LATEST_URL) 和版本标签 (TAG_NAME)
# 3. 将版本标签写入到 version.txt 文件中，供下一阶段使用
RUN JSON_DATA=$(curl -s "https://api.github.com/repos/XIU2/CloudflareSpeedTest/releases/latest") && \
    LATEST_URL=$(echo "$JSON_DATA" | jq -r ".assets[] | select(.name | contains(\"linux_${TARGETARCH}\")) | .browser_download_url") && \
    TAG_NAME=$(echo "$JSON_DATA" | jq -r ".tag_name") && \
    echo "Downloading baked-in version: ${TAG_NAME}" && \
    echo "From URL: ${LATEST_URL}" && \
    curl -L -o cfst.tar.gz "${LATEST_URL}" && \
    echo "${TAG_NAME}" > /download/version.txt

# 解压下载的压缩包
RUN tar -zxvf cfst.tar.gz


# === Stage 2: Go Builder ===
# (此阶段保持不变)
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o test-client ./cmd/main.go

# ... (Stage 1 和 Stage 2 保持不变) ...

# === Stage 3: Final Image ===
FROM alpine
RUN apk add --no-cache ca-certificates curl tar gzip bash tzdata 
ENV TZ=Asia/Shanghai
WORKDIR /app
RUN mkdir -p /app/config

# [核心修正] 确保这里拷贝的是名为 "cfst" 的文件
COPY --from=downloader /download/cfst /usr/local/bin/CloudflareSpeedTest

# ... (后续指令保持不变) ...
COPY --from=downloader /download/ip.txt /app/config/ip.txt
COPY --from=downloader /download/ipv6.txt /app/config/ipv6.txt
COPY --from=downloader /download/version.txt /usr/local/bin/CloudflareSpeedTest.version
COPY --from=builder /app/test-client .

ENTRYPOINT ["./test-client"]