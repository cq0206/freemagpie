# FreeMagpie 架构方案 v1.0

> 基于上游源码审计（2026-11-13, freemagpie 分支）落地的文件级架构。
> 上游仓库结构保持不动（见 FREEMAGPIE.md 的合并原则）；本仓库只做桌面端增强，
> 云端与网站放独立仓库。

## 0. 仓库布局（对 PRD 的修正）

| 仓库 | 内容 | 可见性 |
| --- | --- | --- |
| `cq0206/freemagpie` | Magpie fork（桌面端），目录与上游一致 | **开源**（本仓库） |
| `cq0206/freemagpie-site` | freemagpie.com 静态站（GitHub Pages，已存在） | **私有** |
| `cq0206/freemagpie-cloud` | Go Cloud API + Credits Engine + worker（待建） | 建议私有 |
| `cq0206/freemagpie-web` | Cloudflare Workers 福利门户（待建，见 §5） | 建议私有 |

开源/私有边界的约束：

1. **本仓库（开源）不包含任何云端业务逻辑**：福利数据、核验流程、爬虫、
   管理后台都在私有仓库。开源侧只有「云端目录的客户端」
   （`internal/freemagpie/`），它只讲公共 API，不含内部实现。
2. **API 契约双向解耦**：私有云端和私有网站可以随时改，只要公共 API
   （`/offers`、`/me/…`、`/auth/device/…`）稳定。公共 API 的 spec
   建议在本仓库 `docs/freemagpie/API.md` 维护一份（开源侧看得到的就是
   这个客户端依赖的契约）。
3. **密钥与凭据**：开源仓库绝不含云端部署密钥、数据库 URL、OIDC client
   secret。桌面端连云端只依赖用户运行时配置的云 URL（settings）。
4. **PRD/数据库 schema 归私有仓库**：运营策略（数据来源清单、核验规则、
   合作方）属商业敏感内容，放 `freemagpie-cloud` 内部文档。

理由：magpie 上游每天多个提交、56+ tag/周（见 LESSONS.md）。把 fork 挪进
`apps/desktop/` 意味着每次 merge 全目录冲突。PRD 的 monorepo 结构只适用于
`freemagpie-cloud`（`services/`、`migrations/` 等目录在那里照建）。

## 1. 桌面端改造总览（本仓库）

| 模块 | 实现方式 | 改动面 |
| --- | --- | --- |
| Free Credits Market 页 | 新内置插件 `plugins/credits` | 新增目录，`plugins/all` 加 1 行 |
| 每日签到/福利任务 | 扩展 `plugins/warmups`（Go 侧已有 checkin 框架） | 小 |
| 云端客户端 | 新 `internal/freemagpie/` 包（纯客户端，不碰 gateway） | 新增包 |
| Add to Magpie | 复用 `magpie://import` link + `provider.ParseImport` | 接近零 |
| Free First 路由 | 复用 `routes.FreeFirst` / `FreeExpiring` + 新分组模板 | 新模板 1 个 |
| 配额元数据上传 | 读 `provider.KeptQuotaReport()`，经 `internal/freemagpie` 上报 | 新增 |
| 品牌 | 照 FREEMAGPIE.md 改动清单执行 | 文案级 |

### 1.1 新内置插件 `plugins/credits`

对照 `plugins/usage/usage.go` 的最小形态：

```go
// plugins/credits/credits.go
package credits

import "github.com/yetone/magpie/internal/pluginrt"

func init() {
    pluginrt.Register(pluginrt.Manifest{
        ID: "credits", API: "^1", Requires: []string{"window", "settings"},
        Name: "Free Credits Market", Sub: "Discover, claim and track free AI credits",
        One:  "1 credit offer saved", Many: "{n} credit offers saved",
        InUse: func() (int, error) { /* 读本地 freemagpie.json 的 saved/claimed 数 */ },
        Parts:  []pluginrt.Part{{Slot: "nav.page", Key: "credits"}},
        SetAt:  "network", // Settings 里的云端 URL / 登录开关
        Client: []string{"i18n.js", "credits.css", "client.js"},
    })
}
```

页面文件放 `internal/gui/assets/plugins/credits/`（kernel.js 的 `magpie.plugin("credits", …)`
注册，`slots.add("page", …)`，错误边界和 Try again 由窗口内核托管——见
docs/subsystems/plugin-runtime.md「窗口内核」节）。

**为什么不用社区插件页面**：社区页面在 iframe 沙箱里，CSP `connect-src 'none'`
（plugin-runtime.md P4 节），无法请求外部 API。内置插件页面与 app.js 同源，
经 Go 侧 `/api/` 转发即可访问云端。

**Go 侧 API**：新建 `internal/gui/credits.go`，在 `internal/gui/api.go` 的
路由组装处（`builtinRoutes(mux)` 一带）加 `creditsRoutes(mux, w)` 一行。
handler 只做 `internal/freemagpie` 客户端的薄封装，模式照抄
`internal/gui/importlink.go`（缓存 + `writeJSON`）。

### 1.2 云端客户端 `internal/freemagpie/`

```
internal/freemagpie/
    client.go      // HTTP 客户端（云 URL 来自 settings，走 netproxy 全局代理）
    auth.go        // 设备授权流程（POST /auth/device/code → 轮询 token）
    catalog.go     // GET /offers /products，内存缓存 + 落盘 ~/.config/magpie/freemagpie-cache.json
    claims.go      // 领取事件 push/pull（幂等键）
    quota.go       // 读 provider.KeptQuotaReport() → 脱敏快照 → 定时上传（默认关）
    state.go       // 本地 freemagpie.json（saved/claimed 状态，云端同步源）
```

硬约束（照 AGENTS.md 精神）：
- 该包**不 import** `internal/gateway`，不参与请求路径。
- 云端不可达时所有方法返回缓存/错误，页面照常打开（Cloud 不在关键路径上）。
- 上传内容仅限 `provider.Quota` 结构的脱敏子集（provider/user 名、窗口
  百分比、重置时间），**永不含** key、token、URL。

### 1.2.1 不登录与离线的降级行为（正式契约）

**登录买的是"同步"，不是"使用权"。** 桌面端永远不因未登录、未联网或
Cloud 不可用而减少本地功能：

| 功能 | 不登录 | 登录 |
| --- | --- | --- |
| Magpie 全部原有功能（网关/Provider/Agent/路由/配额/插件） | ✅ 不受任何影响 | ✅ |
| 浏览福利目录、搜索筛选 | ✅ 匿名拉取公共目录 API，缓存离线可看 | ✅ |
| Claim Now（打开官方领取页） | ✅ 浏览器跳转 | ✅ |
| Add to Magpie（导入 provider） | ✅ 本地 `magpie://import` 流程 | ✅ |
| 本地签到（WorkBuddy/Trae/MiniMax/Qoder/ZCode） | ✅ `checkin.go` 本地执行 | ✅ |
| 领取状态记录（跨设备）、云端提醒、Daily Rewards 云端任务 | ❌ | ✅ |

实现要点：
- `internal/freemagpie/catalog.go` 的目录拉取**不携带凭证**（公共 API），
  未登录时照常工作；`claims.go` / 云端提醒在未登录时只做本地记录
  （`state.go` 的 freemagpie.json），登录后补同步。
- 未登录时 Free Credits Market 页面在需要账户的功能处显示登录引导，
  不弹强制登录墙。
- 验收对应 PRD 场景 5（Cloud 不可用）与本节表格：任何一项本地功能
  因未登录而不可用即为 bug。

### 1.3 Add to Magpie（零核心改动路径）

已有机制（`internal/provider/importurl.go:12`）：

```
magpie://import?name=Example&chat=https://api.example.com/v1&models=m1,m2&catalog=example&website=…&icon=…&balance=…&balance_path=…
```

- 网站福利页"Add to Magpie"按钮 = 生成上述链接（`api_compatible=true` 的
  offer 才显示）。
- 用户点击 → 系统唤起 FreeMagpie → `ImportLink` → `stashImport` →
  Providers 页确认对话框（`GET /api/import/{id}`，读一次即焚）→
  用户本地粘贴 key → `POST /api/provider/{action}` 保存。
- key 永不出现在链接里（`key=` 参数可选，网站一律不填）。
- PRD 的 Provider Manifest JSON 字段（name/protocol/base_url/models）到
  import link 参数的映射：`protocol:openai→chat=`，`anthropic→anthropic=`，
  `gemini→gemini=`。manifest 校验逻辑（G-02）在云端做，桌面端只认
  `ParseImport` 已验证的链接。

### 1.4 Free First 路由

`routes.Builtin` 已含 `FreeFirst`（`free`，按模型免费优先）和 `FreeExpiring`
（`expiring`，免费额度先到期先用，group-only）。改造点只有一个：
`internal/provider/grouptemplate.go` 加一个模板（照 `plan`/`thrifty` 的写法），
成员 = 用户标记 `Provider.CountFree` 的 provider，routing=`expiring`。
模板出现在 Routing 页的 Templates 菜单里，无需 UI 改动。

### 1.5 配额元数据同步

数据源：`provider.KeptQuotaReport(time.Now())`（`internal/provider/quotas.go:178`）
——缓存读数、不请求厂商、结构体无凭据。`internal/freemagpie/quota.go` 定时
（默认 6h，可关）取子集上传。PRD 的 `quota_snapshots` 表对应此数据。

### 1.6 Daily Rewards

`internal/provider/checkin.go` 已是通用签到框架（WorkBuddy/Trae/MiniMax/
Qoder/ZCode 各自实现）。桌面端改动：`plugins/credits` 页面读两路数据合并展示：
1. 本地签到状态（`checkin` 框架的结果，经新 `/api/credits/checkins`）；
2. 云端目录里 `reset_period=daily` 的 native credits offer（Claim Now 打开
   官方链接 + Remind Me）。

## 2. 云端（freemagpie-cloud 仓库，概要）

照 PRD PART V 的 schema 落地（users/products/offers/offer_sources/
offer_verifications/user_offer_states/claim_events/quota_snapshots/…），
加 PRD 第 34 节建议的 `user_identities`（(issuer, subject) 唯一）。
技术栈：Go + chi + PostgreSQL + Redis + Asynq，模块化单体，API 与 worker
两个进程。鉴权 OIDC（Web 用 PKCE，桌面用 Device Flow）。

## 3. 同步边界（与上游现有 sync 的关系）

`internal/davsync`/`internal/backup` 是**加密备份**（providers/settings/profiles），
FreeMagpie Cloud 是**账户级福利数据同步**，两者并存互不接管（PRD 第 34 节
第 2 条）。云端同步的实体：offer_state、claim_events、notification_preferences。
设备注册/撤销照 PRD PART XIV D-03~05。

## 4. 实施顺序（Sprint 0 修订）

开源侧（本仓库，可公开开发）：
1. `internal/freemagpie/client.go` + stub 云端（本地 mock server 起步）
2. `plugins/credits` 最小页面（读 mock 目录，搜索/筛选）
3. `docs/freemagpie/API.md`：公共 API 契约（开源侧唯一对云端的依赖物）
4. import link 全链路打通（网站静态页 → 桌面端确认 → 保存）
5. `warmups` 扩展 + Daily Rewards 合并视图
6. 配额上传（opt-in）+ Free First 模板
7. 品牌改动按 FREEMAGPIE.md 清单

私有侧（freemagpie-cloud / freemagpie-web，与开源侧并行）：
a. cloud 仓库 scaffold + 第一批 migration（PRD PART V 的 SQL）
b. 公共目录 API 实现到 API.md 契约
c. web 前台（Discover/详情/搜索），读同一套目录 API
d. 采集/核验 worker（数据来源清单与核验规则留在私有仓库）

开源侧第 1-2 步用 mock server，与私有侧 a-b 无依赖，可同时进行；
第 4 步联调前 API.md 必须定稿。

每步的验收：`make test` 全绿 + 新增测试遵循 docs/code-standards.md
（先失败后通过、沙箱 HOME、`gofmt -l`、跨 OS vet）。

## 5. freemagpie-web：照 usemagpie.ai 的架构做（2026-11-13 补充）

usemagpie.ai 源码在上游 `site/`（本仓库，MIT），已审计。其架构：

- **Cloudflare Workers + 纯静态 HTML**：手写页面（零框架、零构建），
  worker.js（1238 行）前置代理静态资源并提供动态 API；
- **存储全在 CF 边缘**：R2（发布文件）、KV（渠道账本）、D1（插件统计），
  cron 每 5/15 分钟从 GitHub 同步；
- **i18n 是服务端的**：页面标 `data-i18n="key"`，worker 在 `/<lang>/`
  路径把 `i18n.js` 词典填进 HTML；
- **双主题**：首屏内联脚本设 `<html data-theme>` + CSS 变量；
- **统计**：PostHog 经 worker `/r/...` 中继。

FreeMagpie Web 采用同一架构，**不用 Next.js**（对 PRD 技术栈的修正）：

```
freemagpie-web/
    wrangler.jsonc      # Workers 部署（照 site/wrangler.jsonc）
    worker.js           # 路由：静态资源 + /api/catalog/*（读 D1）+ /go 计数跳转
    i18n.js             # data-i18n 词典（en/zh 起步）
    sync.js             # cron：从 freemagpie-cloud 拉取公开目录 → 写 D1
    public/
        index.html      # Discover 首页（骨架、双主题、i18n 标记照搬上游）
        agents/…        # Agent Directory 页
        free-api/…      # Free API 目录页
        offer/…         # 福利详情页骨架
        docs/…          # 帮助文档
```

职责切分（关键决策）：

| 层 | 承担方 | 内容 |
| --- | --- | --- |
| 公共目录（匿名可浏览） | **freemagpie-web（CF Workers + D1）** | Discover、Agent Directory、Free API、详情页、搜索筛选 |
| 目录数据权威源 | **freemagpie-cloud（Go）** | 采集、核验、管理后台；cron 推送到 web 的 D1 |
| 账户数据（登录后） | **freemagpie-cloud（Go）** | 注册登录、领取记录、设备同步、通知 |

理由：usemagpie.ai 已证明 CF Workers 静态 + 边缘 D1 足以承载目录展示，
且运维成本近零；而账户/同步是桌面端与网站共用能力，必须留在 Go 云端
（在 Workers 里写 OIDC device flow 与同步队列不现实）。web 的 D1 是
**只读副本**，权威数据永远在 Go 云端，cron 单向同步——这也让 web 仓库
可以独立开源而不暴露运营数据管道。

“Add to Magpie” 在 web 侧就是生成 `magpie://import?…` 链接（见 §1.3），
纯前端行为，无需任何后端参与。

上游 `site/` 的可复用资产（MIT）：CSS 变量双主题体系、i18n 机制、
docs.css/docs.js、`/go/` 计数跳转、PostHog 中继。

## 6. 公共 API 契约

见 [API.md](API.md)——开源侧与私有云端的唯一契约（目录【anon】/设备流/
import link/用户数据/同步/配额快照）。开源侧的 mock server 与私有云端的
实现都必须满足它；两条开发线的联调以此为冻结点。
