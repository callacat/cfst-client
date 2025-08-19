# === 构建阶段 ===
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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
