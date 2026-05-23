# ── 构建阶段 ──
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# 利用 Docker 缓存层：依赖不变时不重新下载
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建静态二进制，去除调试信息
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w -extldflags '-static'" -o /shopmall .

# ── 运行阶段 ──
FROM alpine:3.19

RUN apk add --no-cache wget ca-certificates tzdata

# 非 root 用户运行
USER nobody

WORKDIR /app
COPY --from=builder /shopmall .
COPY --from=builder /build/config.yaml .

ENV TZ=Asia/Shanghai

EXPOSE 7000

# 等待 MySQL/Redis 就绪后启动，最多重试 30 次（间隔 2s = 60s）
ENTRYPOINT ["sh", "-c", "for i in $(seq 1 30); do ./shopmall && exit 0; echo \"[$(date +%T)] waiting for db/redis, retry $i/30...\"; sleep 2; done; echo 'max retries reached'; exit 1"]
