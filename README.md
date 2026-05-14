# 📸 照片打印商城后端

> 网店下单打印照片系统 – 支持小程序静默登录、照片上传、后台订单管理、管理员登录认证、一键 Docker 部署，并提供热重载开发环境。

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
- 🔐 **双角色认证** – 基于 JWT 的管理员登录 **与** 小程序静默登录（微信 openid）
- 👥 **用户分表** – 后台管理员（`admins`）与小程序用户（`wx_users`）数据隔离，权限清晰
- 🖥️ **后台面板** – 带登录界面的 HTML 管理页，未登录自动跳转
- 📄 **Swagger 文档** – 自动生成 API 文档，在线调试
- 🐳 **Docker 一键运行** – 生产环境与开发环境分离，开箱即用
- ♻️ **热重载开发** – 代码修改自动重启，提升开发效率

---

## 🧰 技术栈

| 组件           | 技术                                                         |
| -------------- | ------------------------------------------------------------ |
| 语言           | Go 1.26.3                                                    |
| Web 框架       | [Gin](https://github.com/gin-gonic/gin)                      |
| ORM            | [GORM](https://gorm.io/) + MySQL 驱动                        |
| 数据库         | MySQL 8.0                                                    |
| 认证           | JWT (golang-jwt/jwt) + bcrypt (管理员) / 微信 openid (小程序) |
| 微信小程序     | 静默登录，通过 `code` 换取 `openid`                          |
| API 文档       | [Swaggo](https://github.com/swaggo/swag) + Swagger UI        |
| 容器化         | Docker + Docker Compose                                      |
| 热重载开发     | [Air](https://github.com/cosmtrek/air)                       |
| 文件存储       | 本地磁盘 (`./uploads`)                                       |

---

## 📁 项目结构

```
photo-print-backend/
├── main.go                 # 入口、路由、Swagger 注解
├── go.mod / go.sum         # 依赖管理
├── Dockerfile              # 生产环境镜像
├── Dockerfile.dev          # 开发环境镜像（集成 Air）
├── docker-compose.yml      # 生产环境编排
├── docker-compose.dev.yml  # 开发环境编排（热重载）
├── .air.toml               # Air 配置文件（可选）
├── .env.example            # 环境变量模板
├── config/                 # 配置加载
├── database/               # 数据库连接 & 迁移
├── models/                 # 数据模型（Admin, WxUser, Photo, Order, OrderItem）
├── controllers/            # 业务控制器（admin_auth, wx_auth, upload, order, photo）
├── middleware/             # 中间件 (CORS, JWT, 角色鉴权)
├── utils/                  # 辅助函数 (响应, 文件, JWT)
├── uploads/                # 上传的照片存储目录
├── static/                 # 后台管理页面
└── docs/                   # Swagger 文档
```

---

## 🚀 快速开始（生产模式）

### 前置条件

- Docker & Docker Compose

### 1️⃣ 克隆项目

```bash
git clone https://github.com/your-username/photo-print-backend.git
cd photo-print-backend
```

### 2️⃣ 配置微信小程序（可选，如需小程序登录）

在 `docker-compose.yml` 或环境变量中设置：

```yaml
environment:
  WECHAT_APP_ID: wx1234567890abcdef
  WECHAT_APP_SECRET: your_wechat_app_secret
```

### 3️⃣ 启动服务

```bash
docker-compose up -d --build
```

该命令会：
- 启动 MySQL 8.0 容器（数据持久化）
- 构建 Go 后端生产镜像（最终只包含二进制文件）
- 挂载 `./uploads` 目录保存照片
- 后端服务暴露 `8080` 端口

### 4️⃣ 初始化管理员

首次启动自动创建默认管理员账户：

- **用户名**: `admin`
- **密码**: `admin123`

> ⚠️ 生产环境请务必修改默认密码！

### 5️⃣ 访问服务

- 后台管理：`http://localhost:8080/admin`
- Swagger 文档：`http://localhost:8080/swagger/index.html`
- 健康检查：`http://localhost:8080/api/v1/photos`

### 6️⃣ 停止服务

```bash
docker-compose down
# 同时删除数据卷
docker-compose down -v
```

---

## 🧪 开发模式（热重载 + 实时日志）

为了方便本地开发，我们提供了独立的开发环境配置文件，支持**代码修改自动重启**，无需手动重新构建镜像。

### 使用步骤

1. **安装 Docker Desktop**（确保 docker-compose 插件可用）

2. **启动开发环境**

```bash
docker-compose -f docker-compose.dev.yml up -d --build
```

3. **查看实时日志**

```bash
docker-compose -f docker-compose.dev.yml logs -f backend
```

4. **修改代码**  
   编辑任意 `.go` 文件，保存后 Air 会自动检测变化 → 重新编译 → 重启服务。  
   无需重新构建镜像或重启容器。

5. **访问服务**  
   同生产模式：`http://localhost:8080/admin`

6. **重启开发环境**

```bash
docker-compose -f docker-compose.dev.yml restart
```

7. **停止但不删除数据**

```bash
cker-compose -f docker-compose.dev.yml down
```

8. **停止并彻底清空数据（重置环境）**

```bash
docker-compose -f docker-compose.dev.yml down -v
```

### 开发环境工作原理

- `docker-compose.dev.yml` 挂载了当前项目目录到容器内的 `/app`
- 容器内运行 [Air](https://github.com/cosmtrek/air) 进程，监听文件变化
- 任何 `.go` 文件改动都会触发 `go build` 并重启进程
- 数据库使用独立的 Docker 卷，数据不会丢失

### 自定义 Air 配置（可选）

在项目根目录创建 `.air.toml` 文件可覆盖默认行为。示例：

```toml
# .air.toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go", "tpl", "html"]
  exclude_dir = ["assets", "tmp", "vendor", "docs"]
  delay = 1000
  stop_on_error = true
```

---

## 🗄️ 数据库访问

你可以通过以下方式进入 MySQL 容器并查看数据表。

### 方式一：通过 Docker 容器进入（推荐）

如果你使用 `docker-compose` 启动的服务，数据库容器名称通常是 `photo-db`（生产环境）或 `photo-db-dev`（开发环境）。

#### 1. 查看容器名称
```bash
docker compose ps
# 或开发环境
docker-compose -f docker-compose.dev.yml ps
```

#### 2. 进入容器内的 MySQL 客户端
```bash
# 生产环境（默认密码 123456）
docker exec -it photo-db mysql -uroot -p123456

# 开发环境（容器名可能为 photo-db-dev）
docker exec -it photo-db-dev mysql -uroot -p123456
```
> 请根据 `docker-compose.yml` 中 `MYSQL_ROOT_PASSWORD` 的实际设置修改密码。

#### 3. 切换数据库并查看表
```sql
USE photoprint;
SHOW TABLES;
```
输出示例：
```
+---------------------+
| Tables_in_photoprint|
+---------------------+
| admins              |
| wx_users            |
| photos              |
| orders              |
| order_items         |
+---------------------+
```

#### 4. 查询具体表内容
```sql
SELECT * FROM admins;                         -- 后台管理员
SELECT * FROM wx_users;                       -- 小程序用户
SELECT * FROM photos ORDER BY created_at DESC LIMIT 10;
SELECT * FROM orders ORDER BY created_at DESC;
SELECT * FROM order_items;
```

#### 5. 退出客户端
```sql
EXIT;
```

### 方式二：使用宿主机 MySQL 客户端（如果映射了 3306 端口）

如果你的 `docker-compose.yml` 中 `db` 服务配置了 `ports: - "3306:3306"`，可以使用本机 MySQL 客户端连接：
```bash
mysql -h 127.0.0.1 -P 3306 -uroot -p123456
```

### 方式三：使用图形化工具（如 TablePlus、Navicat、DBeaver）

连接信息：
- **Host**: `localhost` 或 `127.0.0.1`
- **Port**: `3306`（或你映射的端口）
- **User**: `root`
- **Password**: `123456`（根据实际配置修改）
- **Database**: `photoprint`

---

## 📡 API 接口概览

### 公开接口（无需认证）

| 方法 | 路径                     | 说明                               |
| ---- | ------------------------ | ---------------------------------- |
| POST | `/api/v1/admin/login`    | 后台管理员登录（返回 admin token） |
| POST | `/api/v1/wx/login`       | 小程序静默登录（返回 wx token）    |

### 小程序专用接口（需要 Bearer Token（wx））

| 方法 | 路径                     | 说明                         |
| ---- | ------------------------ | ---------------------------- |
| POST | `/api/v1/upload`         | 上传照片                     |
| POST | `/api/v1/orders`         | 创建订单                     |
| GET  | `/api/v1/orders/:id`     | 查询订单详情                 |

### 后台管理接口（需要 Bearer Token（admin））

| 方法 | 路径                          | 说明               |
| ---- | ----------------------------- | ------------------ |
| GET  | `/api/v1/orders`              | 订单列表（分页）   |
| PUT  | `/api/v1/orders/:id/status`   | 更新订单状态       |
| GET  | `/api/v1/photos`              | 照片列表（分页）   |

> 访问受保护接口时，必须在请求头中添加 `Authorization: Bearer <token>`。  
> 小程序登录和后台管理员登录返回的 token **不可混用**（因为 token 中包含了用户类型，中间件会校验权限）。

### 页面访问

| 路径       | 说明                     |
| ---------- | ------------------------ |
| `/admin`   | 后台管理页面（需登录）   |
| `/swagger/*` | Swagger UI 文档         |
| `/uploads/*` | 访问已上传的照片文件     |

完整交互式文档请访问 `http://localhost:8080/swagger/index.html`。

---

## 🔐 小程序登录流程说明

1. 小程序端调用 `wx.login` 获取 `code`。
2. 调用 `POST /api/v1/wx/login` 发送 `code`。
3. 后端使用 `code` + `WECHAT_APP_ID` + `WECHAT_APP_SECRET` 换取 `openid`。
4. 在 `wx_users` 表中查找或创建该 `openid` 对应的用户记录。
5. 生成 JWT token（`user_type=wx`）并返回。
6. 小程序将 token 存入本地存储，后续请求一律携带 `Authorization: Bearer <token>`。

> 后台管理员登录使用独立的 `POST /api/v1/admin/login`，验证 `username/password`，生成的 token 中 `user_type=admin`。两种 token 通过中间件和角色分组实现路由隔离。

---

## 🐳 环境变量配置

### 生产环境 (`docker-compose.yml`)

| 变量名               | 默认值               | 说明                                 |
| -------------------- | -------------------- | ------------------------------------ |
| `DB_HOST`            | `db`                 | MySQL 主机名                         |
| `DB_PORT`            | `3306`               | 端口                                 |
| `DB_USER`            | `root`               | 用户名                               |
| `DB_PASSWORD`        | `123456`             | 密码                                 |
| `DB_NAME`            | `photoprint`         | 数据库名                             |
| `SERVER_PORT`        | `8080`               | 后端监听端口                         |
| `JWT_SECRET`         | `your-secret-key`    | JWT 签名密钥（生产必须修改）         |
| `WECHAT_APP_ID`      | (空)                 | 微信小程序 AppID（小程序登录需要）   |
| `WECHAT_APP_SECRET`  | (空)                 | 微信小程序 AppSecret（小程序登录需要）|

### 开发环境 (`docker-compose.dev.yml`)

开发环境同样支持以上变量，且额外挂载了源码目录。

---

## 🔒 安全注意事项

- **生产环境务必修改 `JWT_SECRET`**：生成一个强随机字符串（例如 32 位以上）。
- **修改默认管理员密码**：首次登录后请立即更改密码（可通过数据库直接更新）。
- **保护微信小程序密钥**：`WECHAT_APP_SECRET` 绝对不要放在前端代码中，仅在后端环境变量中使用。
- **启用 HTTPS**：在生产环境使用 Nginx 反向代理并配置 SSL 证书。
- **数据库连接**：不要将数据库暴露在公网，使用云数据库的内网地址连接。
- **文件上传限制**：代码已限制单文件最大 5MB，仅允许 jpg/png 格式。

---

## 🤝 贡献

欢迎提交 issue 和 pull request。

---

## 📄 许可证

MIT © [wjun]