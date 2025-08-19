# === 构建阶段 ===
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 1. 优先复制 go.mod 和 go.sum 文件
# 这样可以利用 Docker 的层缓存，只要这两个文件不变，就不需要重新下载依赖
COPY go.mod go.sum ./

# 2. 下载依赖
RUN go mod download

# 3. 复制项目所有剩余的源代码
COPY . .

# 4. 编译应用
# CGO_ENABLED=0 是为了静态编译，不依赖 C 库
RUN CGO_ENABLED=0 GOOS=linux go build -o test-client ./cmd/main.go

# === 运行阶段 ===
FROM alpine
# 安装项目运行所需的基础依赖
RUN apk add --no-cache ca-certificates curl tar gzip

WORKDIR /root
# 从构建阶段复制编译好的二进制文件和配置文件
COPY --from=builder /app/test-client /usr/local/bin/test-client
COPY config.yml .

# 设置环境变量，以便在运行时注入
ENV GITHUB_TOKEN="" \
    GITHUB_PROXY=""

# 设置容器启动命令
ENTRYPOINT ["test-client"]