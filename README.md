# CampusMarket（校园二手交易平台）

一款面向高校学生的校内 C2C 交易平台，覆盖闲置物品发布、价格协商私信、交易达成确认、信誉评分举报、毕业季专场与书籍交换等场景，并支持**商品面交时段预约闭环**：卖家发布商品时可设置若干半小时面交时段，买家下单必须选择未过期且未被占用的时段，待确认期取消自动释放，完成交易后永久锁定。

## 面交时段预约闭环

| 环节 | 行为 |
| --- | --- |
| 卖家发布 | 发布商品时可选挂载若干个整点/半点开始、时长 30 分钟的未来时段（`slots`，可不传） |
| 买家下单 | 商品有时段时必须传 `slot_id`，且该时段须未过期、未被占用；无时段商品维持原有直接下单流程 |
| 并发抢约 | 下单在单个数据库事务内完成「建单 + 时段条件更新」，`WHERE status IN ('available','released') AND start_time > NOW()` 的原子 UPDATE 配合 `(product_id,start_time)`、`order_id` 唯一索引，保证同一时段并发抢购仅一人成功，失败者事务回滚、不留下重复订单 |
| 取消释放 | 仅 `pending`（待确认）阶段买/卖任一方可取消，订单置为 `cancelled` 且时段回到 `released`（可被再次预约） |
| 完成锁定 | 买家确认 → 卖家确认收款后订单 `completed`、商品售出，时段保持 `locked` 永久锁定，不再释放 |
| 详情展示 | 商品详情返回全部时段及 `available_slots` 剩余可选数；「我的交易」展示订单预约时间，刷新后状态一致 |

时段状态 `SlotStatus`：`available`（可预约）/ `locked`（已被订单占用）/ `released`（待确认订单取消后释放、可再约）。


## 快速启动（Docker Compose 一键部署）

```bash
cp .env.example .env
docker compose up -d --build
```

启动后访问：

- 前端：http://localhost:28514
- 后端健康检查：http://localhost:29514/healthz
- MySQL：localhost:3306

预置账号（database/init.sql 种子数据）：

| 角色 | 手机号 | 密码 |
| --- | --- | --- |
| 学生（东校区） | 13700000001 | 123456 |
| 学生（西校区） | 13700000002 | 123456 |
| 学生（南校区） | 13700000003 | 123456 |
| 管理员 | 13800000001 | admin123 |

停止并清理（删除数据卷）：

```bash
docker compose down -v --remove-orphans
```

## 本地开发

后端（Go 1.22）：

```bash
cd backend
go mod tidy
go run ./cmd/server
go build ./...
go test ./...
```

前端（Vue 3 + Vite）：

```bash
cd frontend
npm install
npm run dev
npm run build
```

本地开发时前端 Vite 将 `/api` 代理到 `http://localhost:29514`。

## 技术栈

| 端 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Element Plus + Vite |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 认证 | JWT（golang-jwt/jwt/v5）+ RBAC + bcrypt |
| 其他 | go-playground/validator/v10、log/slog、gin-contrib/cors |

## 项目目录结构

```
cy-354/
├── docker-compose.yml
├── .env.example
├── README.md
├── database/
│   └── init.sql             # MySQL 首启初始化（建表 + 种子数据）
├── backend/
│   ├── go.mod
│   ├── Dockerfile
│   ├── cmd/server/          # main.go + seed.go
│   └── internal/
│       ├── config/          # 环境变量配置
│       ├── constants/       # product.go, trade.go, trade_slot.go, user.go, error_codes.go, log_templates.go, messages.go
│       ├── model/           # user, product, conversation, message, trade_order, trade_slot, review, book_exchange
│       ├── repository/      # GORM 仓库（按实体分文件，含 trade_slot_repository）
│       ├── service/         # 业务逻辑（按实体分文件，含 trade_slot_service）
│       ├── handler/         # HTTP 处理器（按实体分文件）
│       ├── router/          # router.go + 按实体路由文件
│       ├── middleware/      # auth, rbac, rate_limiter, error_handler, request_id
│       ├── dto/             # 请求/响应结构体（含 trade_slot 时段与商品详情视图）
│       └── util/            # jwt, logger, formatters, app_error, credit_calculator, response
└── frontend/
    ├── Dockerfile
    ├── nginx.conf
    └── src/
        ├── api/             # user, product, conversation, tradeOrder, review, bookExchange
        ├── stores/          # authStore, userStore, productStore, tradeStore
        ├── components/common/# ProductCard, ProductForm, ProductSlots, ProductDetailDialog, MessageBubble, TradeStatusBadge, ExchangeCard
        ├── hooks/           # useAuth, useProducts, useConversations
        ├── pages/           # Products, Publish, Messages, Orders, BookExchange, Graduation, Profile, Login, Register
        ├── router/          # index.ts + guards.ts
        ├── utils/           # request, dateFormat（含时段区间 formatSlotRange）, priceFormatter
        ├── constants/       # product, trade, slot, user, errorCodes
        └── types/           # 共享类型（含 TradeSlot / ProductDetail / 订单 slot 字段）
```

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | lpcampusmarket | Compose 项目名/容器前缀 |
| DB_NAME | lpcampusmarket_db | 数据库名 |
| DB_USER | lpcampusmarket_user | 数据库用户 |
| DB_PASSWORD | lpcampusmarket_pwd | 数据库密码 |
| DB_ROOT_PASSWORD | lpcampusmarket_root | root 密码 |
| JWT_SECRET | change_me_to_a_long_random_string | JWT 签名密钥（生产必须修改） |
| JWT_EXPIRE_HOURS | 72 | Token 有效期（小时） |
| RATE_LIMIT_PER_MIN | 120 | 普通接口限流（次/分钟） |
| LOGIN_RATE_LIMIT_PER_MIN | 10 | 登录/注册限流（次/分钟） |
| SEEDING_ENABLED | true | 是否启动时播种数据 |
| CORS_ORIGINS | http://localhost:28514,http://localhost:5173 | 允许跨域来源（逗号分隔；生产严禁 `*`） |
| FRONTEND_PORT | 28514 | 前端端口 |
| BACKEND_PORT | 29514 | 后端端口 |
| DB_PORT | 3306 | MySQL 端口 |

## Docker 部署说明

- 端口映射：前端 `28514:80`，后端 `${BACKEND_PORT:-29514}:8080`，数据库 `${DB_PORT:-3306}:3306`。
- 数据持久化：命名卷 `mysql_data` 挂载到 `/var/lib/mysql`；`database/init.sql` 在首次启动自动执行建表与种子数据。
- 健康检查：db 使用 `mysqladmin ping`，backend 使用 `/healthz`，frontend 依赖 backend healthy。
- 常见问题：
  - 端口冲突：修改 `.env` 中的 `FRONTEND_PORT`/`BACKEND_PORT`/`DB_PORT`。
  - 数据重置：`docker compose down -v` 后重新 `up -d`。
  - 中文目录名：Compose 通过项目名与容器名隔离，任意目录下均可启动。

## API 说明

- 统一前缀 `/api/v1`，健康检查 `/healthz`。
- 响应格式：`{ "code": 0, "message": "ok", "data": ... }`，错误码见 `backend/internal/constants/error_codes.go`。
- 核心接口：
  - `POST /api/v1/users/register`、`POST /api/v1/users/login`、`GET/PUT /api/v1/users/me`
  - `GET/POST /api/v1/products`、`GET/DELETE /api/v1/products/:id`、`GET /api/v1/products/graduation`
  - `POST /api/v1/conversations`、`GET /api/v1/conversations/me`、`GET/POST /api/v1/conversations/:id/messages`
  - `POST /api/v1/trade-orders`、`GET /api/v1/trade-orders/me`、`POST /api/v1/trade-orders/:id/buyer-confirm|seller-confirm|cancel`
  - `POST /api/v1/reviews`、`GET /api/v1/reviews/me`
  - `GET/POST /api/v1/book-exchanges`、`POST /api/v1/book-exchanges/:id/close`
  - `GET /api/v1/admin/stats`（管理员）

## API 接口清单

统一前缀 `/api/v1`；鉴权列中「登录」表示需要 JWT，「管理员」表示需要管理员角色。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/healthz` | 健康检查 | 无 |
| POST | `/api/v1/users/register` | 注册学生账号 | 无（登录限流） |
| POST | `/api/v1/users/login` | 登录获取 JWT | 无（登录限流） |
| GET | `/api/v1/users/me` | 当前用户信息 | 登录 |
| PUT | `/api/v1/users/me` | 更新昵称/头像/校区 | 登录 |
| GET | `/api/v1/products` | 商品分页列表 | 无 |
| GET | `/api/v1/products/graduation` | 毕业季专场列表 | 无 |
| GET | `/api/v1/products/:id` | 商品详情（含面交时段与剩余可选数） | 无 |
| POST | `/api/v1/products` | 发布商品（可携带 `slots` 半小时面交时段） | 登录 |
| DELETE | `/api/v1/products/:id` | 下架自己的商品 | 登录 |
| POST | `/api/v1/conversations` | 发起/复用私信会话 | 登录 |
| GET | `/api/v1/conversations/me` | 我的会话列表 | 登录 |
| GET | `/api/v1/conversations/:id/messages` | 会话消息记录 | 登录 |
| POST | `/api/v1/conversations/:id/messages` | 发送私信 | 登录 |
| POST | `/api/v1/trade-orders` | 创建购买订单（有时段的商品必传 `slot_id`） | 登录 |
| GET | `/api/v1/trade-orders/me` | 我的订单列表 | 登录 |
| POST | `/api/v1/trade-orders/:id/buyer-confirm` | 买家确认 | 登录 |
| POST | `/api/v1/trade-orders/:id/seller-confirm` | 卖家确认（订单完成+商品售出） | 登录 |
| POST | `/api/v1/trade-orders/:id/cancel` | 取消订单 | 登录 |
| POST | `/api/v1/reviews` | 交易后评价（含信誉积分） | 登录 |
| GET | `/api/v1/reviews/me` | 我收到的评价 | 登录 |
| GET | `/api/v1/book-exchanges` | 书籍交换列表 | 无 |
| POST | `/api/v1/book-exchanges` | 发布换书请求（自动匹配） | 登录 |
| POST | `/api/v1/book-exchanges/:id/close` | 关闭换书请求 | 本人 |
| GET | `/api/v1/admin/stats` | 平台统计占位接口 | 管理员 |

## 枚举出现位置清单

### ProductStatus（on_sale/reserved/sold/removed）

前端 `frontend/src/constants/product.ts`：

- `PRODUCT_STATUSES` 常量定义
- `productStatusLabel()` / `productStatusType()` 映射
- `src/components/common/ProductCard.vue` 状态徽章与购买按钮显隐
- `src/pages/Orders.vue` 交易联动

后端 `backend/internal/constants/product.go`：

- `ProductStatusOnSale/Reserved/Sold/Removed` 常量
- `ProductStatuses` 列表、`IsProductStatus()`
- `ProductStatusText()` 文案
- `backend/internal/model/product.go` Status 字段
- `backend/internal/service/product_service.go` 发布/下架/售出状态机
- `backend/internal/util/formatters.go` `ProductStatusText()`
- `backend/internal/constants/log_templates.go` 商品状态日志模板
- `backend/internal/constants/error_codes.go` 状态冲突错误码

### TradeStatus（pending/confirmed/completed/cancelled）

前端 `frontend/src/constants/trade.ts`：

- `TRADE_STATUSES` 常量定义
- `tradeStatusLabel()` / `tradeStatusType()` 映射
- `src/components/common/TradeStatusBadge.vue` 状态徽章
- `src/pages/Orders.vue` 按钮显隐（确认收货/确认收款/取消/评价）

后端 `backend/internal/constants/trade.go`：

- `TradeStatusPending/Confirmed/Completed/Cancelled` 常量
- `TradeStatuses` 列表、`IsTradeStatus()`
- `TradeStatusText()` 文案
- `backend/internal/model/trade_order.go` Status 字段
- `backend/internal/service/trade_order_service.go` 交易状态机
- `backend/internal/util/formatters.go` `TradeStatusText()`
- `backend/internal/constants/log_templates.go` 交易日志模板
- `backend/internal/constants/error_codes.go` 状态冲突错误码

### SlotStatus（available/locked/released，面交时段）

前端 `frontend/src/constants/slot.ts`：

- `SLOT_STATUSES` 常量定义
- `slotStatusLabel()` / `slotStatusType()` 映射
- `src/components/common/ProductSlots.vue` 可选时段单选与剩余数
- `src/components/common/ProductDetailDialog.vue` 商品详情时段展示
- `src/pages/Orders.vue` 订单预约时间与时段状态
- `src/types/index.ts` `TradeSlot.status` 字段

后端 `backend/internal/constants/trade_slot.go`：

- `SlotStatusAvailable/Locked/Released` 常量
- `SlotStatuses` 列表、`IsSlotStatus()`
- `SlotStatusText()` 文案、`SlotDuration`（30 分钟）、`MaxSlotsPerProduct`
- `backend/internal/model/trade_slot.go` Status 字段
- `backend/internal/repository/trade_slot_repository.go` 原子占用 `TryOccupy()` / 释放 `Release()`
- `backend/internal/service/trade_slot_service.go` 时段校验与可用性视图
- `backend/internal/service/trade_order_service.go` 下单锁定/取消释放/完成永久锁定状态机
- `backend/internal/constants/messages.go` 时段相关错误文案
- `backend/internal/constants/log_templates.go` 时段日志模板

### UserRole（student/admin）

前端 `frontend/src/constants/user.ts`：

- `USER_ROLES` 常量定义
- `roleLabel()` 映射
- `src/stores/authStore.ts` `isAdmin()`
- `src/hooks/useAuth.ts` `hasRole()`
- `src/pages/Profile.vue` 角色展示

后端 `backend/internal/constants/user.go`：

- `UserRoleStudent/Admin` 常量
- `UserRoles` 列表、`IsUserRole()`
- `UserRoleText()` 文案
- `backend/internal/model/user.go` Role 字段
- `backend/internal/middleware/rbac.go` 权限校验
- `backend/internal/router/*.go` 路由权限（管理员接口）
- `backend/internal/util/jwt.go` Claims.Role
- `backend/internal/util/formatters.go` `RoleText()`
- `backend/internal/constants/log_templates.go` 登录日志带角色

## 质量说明

- 后端 `go build ./...` 与 `go test ./...` 通过（含 service/util 表驱动单测）。
- 前端 `npm run build` 零错误。
- 分层依赖单向：handler → service → repository → model；构造器注入；`%w` 错误链 + 哨兵错误；统一响应 `{code,message,data}`。
- 日志模板集中于 `internal/constants/log_templates.go`（≥25 条），全栈引用，字段变更需联动修改（屎山设计约束）。

## License

MIT
