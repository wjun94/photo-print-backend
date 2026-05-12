# Photo Print Backend

这是一个使用 Go 语言开发的后端服务，用于处理照片打印相关的订单和文件上传。该项目集成了 Swaggo 以生成 API 文档，并通过 Docker 进行容器化部署。

# 📸 照片打印商城后端

> 网店下单打印照片系统 – 支持小程序上传照片、后台订单管理、一键 Docker 部署。

![Go Version](https://img.shields.io/badge/Go-1.26.3-blue)
![Gin](https://img.shields.io/badge/Gin-1.10.0-lightblue)
![GORM](https://img.shields.io/badge/GORM-1.25.12-green)
![Swagger](https://img.shields.io/badge/Swagger-1.16.3-orange)
![Docker](https://img.shields.io/badge/Docker-20.10+-blue)

---

## ✨ 特性

- 🖼️ **照片上传** – 小程序端直接上传 JPG/PNG，自动保存到服务器
- 🛒 **下单打印** – 选择照片、规格、数量，生成订单
- 📋 **订单管理** – 后台查看订单列表、详情，修改状态（待处理/已支付/处理中/已完成/已取消）
- 🖥️ **后台面板** – 极简 HTML 管理界面，无需额外前端
- 📄 **Swagger 文档** – 自动生成 API 文档，在线调试
- 🐳 **Docker 一键运行** – 包含 MySQL + 后端，开箱即用

---

## 🧰 技术栈

| 组件           | 技术                                                         |
| -------------- | ------------------------------------------------------------ |
| 语言           | Go 1.26.3                                                    |
| Web 框架       | [Gin](https://github.com/gin-gonic/gin)                      |
| ORM            | [GORM](https://gorm.io/) + MySQL 驱动                        |
| 数据库         | MySQL 8.0                                                    |
| API 文档       | [Swaggo](https://github.com/swaggo/swag) + Swagger UI        |
| 容器化         | Docker + Docker Compose                                      |
| 跨域处理       | 自定义中间件                                                  |
| 文件存储       | 本地磁盘 (`./uploads`)                                       |

---

## 📁 项目结构

```
photo-print-backend/
├── main.go                 # 入口、路由、Swagger 注解
├── go.mod / go.sum         # 依赖管理
├── Dockerfile              # 多阶段构建镜像
├── docker-compose.yml      # 一键启动 MySQL + 后端
├── .env.example            # 环境变量模板
├── config/                 # 配置加载
│   └── config.go
├── database/               # 数据库连接 & 迁移
│   └── db.go
├── models/                 # 数据模型 (Photo, Order, OrderItem)
│   └── models.go
├── controllers/            # 业务控制器
│   ├── upload.go
│   ├── order.go
│   └── photo.go
├── middleware/             # 中间件 (CORS)
│   └── cors.go
├── utils/                  # 辅助函数 (响应封装、文件保存)
│   ├── response.go
│   └── file.go
├── uploads/                # 上传的照片存储目录 (挂载卷)
├── static/                 # 静态后台管理页面
│   └── admin.html
└── docs/                   # swag init 生成的文档
    ├── docs.go
    ├── swagger.json
    └── swagger.yaml
```

---

## 🚀 快速开始

### 前置条件

- Docker & Docker Compose
- (可选) Go 1.26.3 本地开发环境、Make

### 1️⃣ 克隆项目

```bash
git clone https://github.com/your-username/photo-print-backend.git
cd photo-print-backend
```

### 2️⃣ 使用 Docker Compose 启动（推荐）

```bash
docker-compose up -d --build
```

该命令会：
- 启动 MySQL 8.0 容器（数据持久化）
- 构建 Go 后端镜像
- 挂载 `./uploads` 到宿主机保存照片
- 后端服务暴露在 `8080` 端口

### 3️⃣ 验证服务

```bash
# 查看容器状态
docker-compose ps

# 测试接口
curl http://localhost:8080/api/v1/photos

# 访问 Swagger 文档
open http://localhost:8080/swagger/index.html

# 访问后台管理页面
open http://localhost:8080/admin
```

### 4️⃣ 停止服务

```bash
docker-compose down
# 如需清理数据卷（删除所有数据）
docker-compose down -v
```

---

## 🧪 本地开发（不使用 Docker）

### 安装 Go 1.26.3

- 官网下载: [https://go.dev/dl/](https://go.dev/dl/)
- 或使用版本管理工具: [g](https://github.com/voidint/g)

### 安装依赖 & 运行

```bash
# 下载依赖
go mod tidy

# 安装 swag 命令行工具（生成文档）
go install github.com/swaggo/swag/cmd/swag@latest

# 生成 Swagger 文档
swag init

# 启动 MySQL（需自行安装或使用 docker run）
docker run -d --name mysql-photo -p 3306:3306 -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=photoprint mysql:8.0

# 设置环境变量
export DB_HOST=localhost DB_USER=root DB_PASSWORD=123456 DB_NAME=photoprint

# 运行后端
go run main.go
```

后端将运行在 `http://localhost:8080`。

---

## 📡 API 接口概览

| 方法 | 路径                           | 说明               |
| ---- | ------------------------------ | ------------------ |
| POST | `/api/v1/upload`               | 上传照片           |
| POST | `/api/v1/orders`               | 创建订单           |
| GET  | `/api/v1/orders`               | 订单列表（分页）   |
| GET  | `/api/v1/orders/:id`           | 订单详情           |
| PUT  | `/api/v1/orders/:id/status`    | 更新订单状态       |
| GET  | `/api/v1/photos`               | 照片列表（分页）   |
| GET  | `/admin`                       | 后台管理页面       |
| GET  | `/uploads/:filename`           | 访问照片文件       |

完整交互式文档：`http://localhost:8080/swagger/index.html`

---

## 🐳 Docker 自定义配置

通过环境变量覆盖默认配置（修改 `docker-compose.yml` 或直接在 `environment` 中定义）：

| 变量名       | 默认值      | 说明               |
| ------------ | ----------- | ------------------ |
| `DB_HOST`    | `db`        | MySQL 主机名       |
| `DB_PORT`    | `3306`      | 端口               |
| `DB_USER`    | `root`      | 用户名             |
| `DB_PASSWORD`| `123456`    | 密码               |
| `DB_NAME`    | `photoprint`| 数据库名           |
| `SERVER_PORT`| `8080`      | 后端监听端口       |

---

## 📜 许可证

MIT © [Your Name]
