# === 构建阶段 ===
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum 文件，并下载依赖
# 这一步可以利用 Docker 的缓存机制，如果 go.mod 没有变化，则不会重复下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有剩余的源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o test-client ./cmd/main.go

# === 运行阶段 ===
FROM alpine
RUN apk add --no-cache ca-certificates curl tar gzip

WORKDIR /root
COPY --from=builder /app/test-client /usr/local/bin/test-client
COPY config.yml .

ENV GITHUB_TOKEN="" \
    GITHUB_PROXY=""

ENTRYPOINT ["test-client"]