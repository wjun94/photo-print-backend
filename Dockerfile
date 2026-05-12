# 构建阶段
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# 安装 swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 生成 swagger 文档
RUN swag init

# 编译
RUN go build -o photo-print-backend .

# 运行阶段（关键：用固定版本，不走 latest 网络拉取）
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/photo-print-backend .

# 复制静态文件（关键！）
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./photo-print-backend"]