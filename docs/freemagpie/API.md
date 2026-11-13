# FreeMagpie 公共 API 契约 v0.1

> 本文件是开源桌面端（`internal/freemagpie/` 客户端）与私有云端
> （freemagpie-cloud）、网站（freemagpie-web 的 D1 副本）之间的**唯一契约**。
> 云端内部实现可随时重构，公共 API 只做**向后兼容的变更**（加字段可以，
> 删字段/改语义/改路径必须升版本路径）。
>
> 版本：v0.1（草案，Sprint 0 定稿）
> Base URL：用户可配（settings，默认 `https://api.freemagpie.com`），
> 路径前缀 `/api/v1`。

## 0. 通用约定

- REST + JSON；UTC ISO 8601 时间戳；UUID 资源 id。
- 列表一律游标分页：`?limit=`（默认 50，最大 200）+ `?cursor=`；
  响应带 `next_cursor`（null 表示到底）。
- 匿名可访问的接口标注【anon】；其余需要 `Authorization: Bearer <token>`。
- 错误体统一：

```json
{ "error": { "code": "OFFER_EXPIRED", "message": "…", "request_id": "req_…" } }
```

- 桌面端对云端的所有请求必须可失败：5xx/超时/断网时回退缓存，不重试
  超过一次（ARCHITECTURE.md §1.2）。

## 1. 目录（匿名只读）

### GET /api/v1/offers【anon】

福利列表，支持筛选。参数：

| 参数 | 值 |
| --- | --- |
| `product` | 产品 slug（`qoder`） |
| `category` | `coding-agent` / `general-agent` / `api-provider` / `model` |
| `reward_kind` | `native_credit` / `api_credit` / `free_model` / `promo` |
| `region` | `CN` / `GLOBAL`（JSONB contains） |
| `api_compatible` | `true` / `false` |
| `reset_period` | `daily` / `weekly` / `monthly` / `none` |
| `status` | 默认 `active`；`expiring`（7 天内到期）/ `expired` 需显式请求 |
| `sort` | `recent`（默认）/ `verified` / `expiring` / `popular` |

响应：

```json
{
  "data": [{
    "id": "uuid",
    "slug": "qoder-daily-credits",
    "product": { "id": "uuid", "slug": "qoder", "name": "Qoder", "icon": "…", "category": "coding-agent" },
    "title": "Daily Credits",
    "reward_kind": "native_credit",
    "reward_label": "按官方公布额度",
    "reset_period": "daily",
    "claim_method": "manual",
    "claim_url": "https://…",
    "region_codes": ["CN"],
    "valid_until": "2026-12-31T00:00:00Z",
    "api_compatible": false,
    "verification_status": "official_verified",
    "last_verified_at": "2026-11-12T04:00:00Z"
  }],
  "next_cursor": null
}
```

**桌面端义务**：`api_compatible=false` 的 offer 永不显示 Add to Magpie。

### GET /api/v1/offers/{slug}【anon】

详情，含 `eligibility`（JSONB）、`sources[]`（证据链接）与
`import_link`（见 §3）。

### GET /api/v1/products【anon】/ GET /api/v1/products/{slug}【anon】

产品目录与详情（独立详情页数据）。

### GET /api/v1/discover/{view}【anon】

`view` = `featured` / `new` / `expiring` / `daily`：网站的四个聚合区，
同一响应结构，供桌面端 Discover 首屏复用。

## 2. 认证（桌面端设备流）

### POST /api/v1/auth/device/code

```json
{ "device_name": "MacBook Pro", "platform": "darwin" }
```

响应（OAuth Device Flow 标准）：

```json
{ "device_code": "…", "user_code": "ABCD-EFGH", "verification_url": "https://freemagpie.com/activate", "expires_in": 600, "interval": 5 }
```

### POST /api/v1/auth/device/token

轮询。`authorization_pending` / `slow_down` / `expired_token` 按 RFC 8628。
成功返回短期 access token + 轮换 refresh token。

设备注册在首次 token 时隐式完成；`GET /api/v1/me/devices`、
`DELETE /api/v1/me/devices/{id}` 管理与撤销。

## 3. Provider 导入（Add to Magpie）

`GET /api/v1/offers/{slug}` 的 `import_link` 字段就是一条
`magpie://import?…` URL（`internal/provider/importurl.go` 已定义参数集）：

```
magpie://import?name=Example&chat=https://api.example.com/v1&models=m1,m2&catalog=example&website=…&icon=…
```

规则（云端生成时必须遵守，桌面端只信任 `provider.ParseImport` 的校验）：

- `key=` 参数**永不携带**——用户在桌面端本地粘贴；
- `protocol` 到参数的映射：openai→`chat=`，anthropic→`anthropic=`，
  gemini→`gemini=`；
- `balance=` 只能是 offer 自己端点同 host 的 https URL（ParseImport 会拒绝其他）。

**桌面端无新增代码路径**：链接经 `ImportLink` → `stashImport` →
Providers 页确认，与上游完全一致。

## 4. 用户数据（登录后）

| 端点 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/me` | GET | 当前用户 |
| `/api/v1/me/offers` | GET | 已保存/已领取的 offer 状态 |
| `/api/v1/me/offers/{id}/state` | PUT | `{state: saved\|claimed\|dismissed, claimed_at}` |
| `/api/v1/me/claims` | GET/POST | 领取事件（append-only，`Idempotency-Key` 头去重） |
| `/api/v1/me/notifications` | GET | 通知列表 |
| `/api/v1/me/notifications/preferences` | PATCH | 偏好 |

**未登录行为**：桌面端在未登录时对 §4 全部端点不发请求；本地状态记在
`freemagpie.json`（`state.go`），登录后由 `/me/sync/push` 补传
（ARCHITECTURE.md §1.2.1 契约）。

### 同步

```
GET  /api/v1/me/sync/pull?cursor=<n>   → {cursor, changes[], has_more}
POST /api/v1/me/sync/push              → {device_id, changes[{idempotency_key, entity_type, entity_id, operation, payload}]}
```

`entity_type` 当前仅 `offer_state`。冲突规则：`claimed_at` 取较早者
（先领先用），`state` 以非 `saved` 者为准。

## 5. 配额快照（opt-in，登录后）

```
POST /api/v1/me/quotas/snapshots
[{ "product_slug": "qoder", "kind": "native_credit",
   "windows": [{"name": "daily", "used": 0.4, "resets_at": "…"}], "observed_at": "…" }]
```

- 桌面端默认**关**，Settings 里显式开启（PRD 隐私原则）；
- 载荷是 `provider.Quota` 的脱敏子集：`windows[].used`（0-1 的份额）、
  `resets_at`、`kind`、`product_slug`。**不含**：key、token、账号邮箱、
  余额 URL、任何字符串形式的凭据（PRD PART IX 23.3）；
- 云端只存最近 N=30 天，超出滚动删除。

## 6. 变更纪律

- 加字段：随时（客户端忽略未知字段）。
- 破坏性变更：新版本前缀 `/api/v2`，v1 至少维护到 v2 发布后 6 个月
  （桌面端用户升级慢，旧版本会长期在线）。
- 本文件的每次变更在同一次提交里说明桌面端 `internal/freemagpie/`
  是否需要跟进。
