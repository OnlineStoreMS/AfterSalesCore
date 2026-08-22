# AfterSalesCore — 售后开箱视频管理

独立应用，与 [UserCore](../UserCore)（IAM）、[ProductCore](../ProductCore)（PIM）、[SupplyCore](../SupplyCore)（供应链）并列部署。

| 组件 | 端口 | 说明 |
|------|------|------|
| API | 8093 | Go + Gin + GORM |
| Web | 5176 | Vue 3 + Element Plus |

## 当前能力

### 开箱视频录制
- 摄像头预览 + 条码扫描识别快递单号（`@zxing/browser`）
- `MediaRecorder` 全程录制开箱视频
- 问题凭证照片拍摄与说明
- 视频/图片上传至 MinIO（或本地存储）

### 开箱记录管理
- 按快递单号搜索开箱记录
- 详情页视频预览、下载与问题照片画廊
- 状态：草稿 → 已完成

### 店铺售后工作台
- 按平台（抖店 / 淘宝 / 拼多多）管理售后店铺
- 生成绑定码，供抖店 Chrome 扩展认领店铺并上报快捷筛选卡片与售后单
- 按店铺查看与抖店类似的售后工作台

## API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/unboxing-records` | 列表（支持 `trackingNo` 筛选） |
| POST | `/api/v1/admin/unboxing-records` | 创建记录 |
| GET | `/api/v1/admin/unboxing-records/:id` | 详情 |
| POST | `/api/v1/admin/unboxing-records/:id/video` | 上传视频 |
| POST | `/api/v1/admin/unboxing-records/:id/photos` | 上传问题照片 |
| POST | `/api/v1/admin/unboxing-records/:id/complete` | 完成记录 |
| GET | `/api/v1/admin/unboxing-records/:id/video/download` | 获取视频下载链接 |
| GET | `/api/v1/admin/shops` | 售后店铺列表 |
| POST | `/api/v1/admin/shops` | 新增店铺 |
| GET | `/api/v1/admin/shops/:id/workbench` | 店铺售后工作台（卡片 + 售后单） |
| POST | `/api/v1/admin/shops/:id/reset-bind` | 重置绑定码 |
| POST | `/api/v1/plugin/bind` | 扩展用绑定码换密钥 |
| POST | `/api/v1/plugin/heartbeat` | 扩展心跳 |
| POST | `/api/v1/plugin/sync` | 扩展上报卡片与售后单 |

抖店插件见 `extensions/doudian-aftersale/`。

## 快速开始

```bash
# 1. 数据库（PostgreSQL）
make init-db APP_PASSWORD=你的密码

# 2. 配置
cp configs/config.example.yaml configs/config.yaml
# jwt_secret 必须与 UserCore 一致

# 3. 后端
make tidy
make run

# 4. 前端
cd web && npm install && npm run dev
```

浏览器访问 http://localhost:5176 ，从 UserCore 应用中心进入（需配置 AfterSalesCore 应用）。

## 数据库权限问题

若启动报错 `relation "unboxing_records" already exists` 或权限相关错误，以超级用户执行：

```bash
chmod +x deploy/fix_db_permissions.sh
./deploy/fix_db_permissions.sh
```

## UserCore 注册

在 UserCore 应用中心应出现 **售后中心**。新环境 seed 会自动写入；已有环境启动 UserCore 时会 `EnsureApps` upsert。

权限码：

- `aftersales:read` — 查看开箱记录
- `aftersales:write` — 录制开箱、上传视频与照片

## 环境变量（前端）

| 变量 | 默认 |
|------|------|
| `VITE_API_GATEWAY` | 未设置时 API 代理到 `http://localhost:8093` |
