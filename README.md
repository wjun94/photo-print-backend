# Photo Print Backend

这是一个使用 Go 语言开发的后端服务，用于处理照片打印相关的订单和文件上传。该项目集成了 Swaggo 以生成 API 文档，并通过 Docker 进行容器化部署。

# 📸 照片打印商城后端

> 网店下单打印照片系统 – 支持小程序上传照片、后台订单管理、**管理员登录认证**、一键 Docker 部署。

![Go Version](https://img.shields.io/badge/Go-1.26.3-blue)
![Gin](https://img.shields.io/badge/Gin-1.10.0-lightblue)
![GORM](https://img.shields.io/badge/GORM-1.25.12-green)
![JWT](https://img.shields.io/badge/JWT-5.2.0-yellow)
![Swagger](https://img.shields.io/badge/Swagger-1.16.3-orange)
![Docker](https://img.shields.io/badge/Docker-20.10+-blue)

---

## ✨ 特性

- 🖼️ **照片上传** – 小程序端直接上传 JPG/PNG，自动保存到服务器
- 🛒 **下单打印** – 选择照片、规格、数量，生成订单
- 📋 **订单管理** – 后台查看订单列表、详情，修改订单状态
- 🔐 **登录认证** – 基于 JWT 的管理员登录，保护后台 API
- 🖥️ **后台面板** – 带登录界面的 HTML 管理页，未登录自动跳转
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
| 认证           | JWT (golang-jwt/jwt) + bcrypt                                |
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
├── database/               # 数据库连接 & 迁移 & 默认管理员初始化
│   └── db.go
├── models/                 # 数据模型 (Photo, Order, OrderItem, User)
│   └── models.go
├── controllers/            # 业务控制器 (新增 auth 控制器)
│   ├── upload.go
│   ├── order.go
│   ├── photo.go
│   └── auth.go
├── middleware/             # 中间件 (CORS, JWT 认证)
│   ├── cors.go
│   └── auth.go
├── utils/                  # 辅助函数 (响应, 文件保存, JWT)
│   ├── response.go
│   ├── file.go
│   └── jwt.go
├── uploads/                # 上传的照片存储目录 (挂载卷)
├── static/                 # 静态后台管理页面
│   └── admin.html          # 带登录界面的管理面板
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
- 构建 Go 后端镜像（自动安装依赖、生成 Swagger 文档）
- 挂载 `./uploads` 到宿主机保存照片
- 后端服务暴露在 `8080` 端口

### 3️⃣ 初始化管理员账户

首次启动时，数据库会自动初始化并创建默认管理员：

- **用户名**: `admin`
- **密码**: `admin123`

> ⚠️ 生产环境请务必修改默认密码！可通过直接修改数据库或调用 API 重置。

### 4️⃣ 验证服务

```bash
# 查看容器状态
docker-compose ps

# 测试公开接口
curl http://localhost:8080/api/v1/photos

# 访问后台管理页面（自动跳转登录）
open http://localhost:8080/admin

# 访问 Swagger 文档
open http://localhost:8080/swagger/index.html
```

### 5️⃣ 登录后台

1. 访问 `http://localhost:8080/admin`
2. 输入用户名 `admin`，密码 `admin123`
3. 登录后即可管理订单和查看所有照片

### 6️⃣ 停止服务

```bash
docker-compose down
# 如需清理数据卷（删除所有数据）
docker-compose down -v
```

---

## 🧪 本地开发（不使用 Docker）

参见 [本地开发指南](LOCAL_DEV.md) 或直接执行：

```bash
# 安装 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 下载依赖
go mod tidy

# 生成 Swagger 文档
swag init

# 运行 MySQL（需提前安装或使用 Docker 独立运行）
docker run -d --name mysql-local -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=photoprint -p 3306:3306 mysql:8.0

# 设置环境变量
export DB_HOST=localhost DB_USER=root DB_PASSWORD=123456 DB_NAME=photoprint SERVER_PORT=8080

# 运行
go run main.go
```

---

## 📡 API 接口概览

### 公开接口（无需认证）

| 方法 | 路径                     | 说明               |
| ---- | ------------------------ | ------------------ |
| POST | `/api/v1/login`          | 管理员登录         |
| POST | `/api/v1/upload`         | 上传照片（小程序） |
| POST | `/api/v1/orders`         | 创建订单           |
| GET  | `/api/v1/orders/:id`     | 订单详情           |

### 受保护接口（需要 Bearer Token）

| 方法 | 路径                          | 说明               |
| ---- | ----------------------------- | ------------------ |
| GET  | `/api/v1/orders`              | 订单列表（分页）   |
| PUT  | `/api/v1/orders/:id/status`   | 更新订单状态       |
| GET  | `/api/v1/photos`              | 照片列表（分页）   |

> 访问受保护接口时，必须在请求头中添加 `Authorization: Bearer <token>`。登录接口会返回 token。

### 页面

| 路径       | 说明                     |
| ---------- | ------------------------ |
| `/admin`   | 后台管理页面（需登录）   |
| `/swagger/*` | Swagger UI 文档         |
| `/uploads/*` | 访问已上传的照片文件     |

完整交互式文档请访问 `http://localhost:8080/swagger/index.html`。

---

## 🐳 Docker 自定义配置

通过环境变量覆盖默认配置（修改 `docker-compose.yml` 或直接在 `environment` 中定义）：

| 变量名           | 默认值               | 说明                       |
| ---------------- | -------------------- | -------------------------- |
| `DB_HOST`        | `db`                 | MySQL 主机名               |
| `DB_PORT`        | `3306`               | 端口                       |
| `DB_USER`        | `root`               | 用户名                     |
| `DB_PASSWORD`    | `123456`             | 密码                       |
| `DB_NAME`        | `photoprint`         | 数据库名                   |
| `SERVER_PORT`    | `8080`               | 后端监听端口               |
| `JWT_SECRET`     | `your-secret-key`    | JWT 签名密钥（生产必须修改）|

在 `docker-compose.yml` 中增加 `JWT_SECRET` 环境变量可覆盖默认密钥。

---

## 🔒 安全注意事项

- **生产环境务必修改 `JWT_SECRET`**：在 `utils/jwt.go` 中或通过环境变量设置一个强随机字符串。
- **修改默认管理员密码**：首次登录后请立即更改密码（可通过数据库直接更新或添加修改密码 API）。
- **启用 HTTPS**：在生产环境使用 Nginx 反向代理并配置 SSL 证书。
- **数据库连接**：不要将数据库暴露在公网，使用云数据库的内网地址连接。
- **文件上传限制**：代码已限制单文件最大 5MB，仅允许 jpg/png 格式。

---

## 🤝 贡献

欢迎提交 issue 和 pull request。

---

## 📄 许可证

MIT © [Your Name]
