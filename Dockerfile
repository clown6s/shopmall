# ── 构建阶段 ──
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# 利用 Docker 缓存层：依赖不变时不重新下载
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建静态二进制，去除调试信息
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w -extldflags '-static'" -o /shopmall .

# ── 运行阶段 ──
FROM scratch

# CA 证书（HTTPS 请求必需）+ 时区数据
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 非 root 用户运行
USER nobody

WORKDIR /app
COPY --from=builder /shopmall .
COPY --from=builder /build/config.yaml .

ENV TZ=Asia/Shanghai

EXPOSE 7000

ENTRYPOINT ["./shopmall"]
