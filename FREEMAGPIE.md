# free-magpie

[freemagpie.com](https://freemagpie.com) — 基于 [yetone/magpie](https://github.com/yetone/magpie)（MIT）的下游发行版。

上游每天有多个提交。本仓库的目标是：**尽量少改，跟上上游**。

## 分支模型

| 分支 | 用途 |
| --- | --- |
| `main` | 只做 fast-forward，永远等于 `upstream/main`，不携带任何本地提交 |
| `freemagpie` | 我们的发行分支。所有自由版改动提交在这里，定期 `git merge main` 同步 |

禁止在 `main` 上直接提交；上游有 force-push 或 rebase 时（发生过），
用 `git fetch upstream && git reset --hard upstream/main` 处理 main。

## remote 配置

- `upstream` → `https://github.com/yetone/magpie.git`
  - 第二个 fetch URL：`https://ghfast.top/https://github.com/yetone/magpie.git`
    （直连 GitHub 不通时的镜像，git 会依次尝试两个 URL）
  - push 已禁用（`no-push-only-fetch`）：上游是只读的，防误推
- `origin` → `git@github.com:cq0206/freemagpie.git`（SSH，22 端口可用）
  - `main` 和 `freemagpie` 都已推送到 origin；上游的 tag 未推（跟踪用本地即可）

## 同步上游（每次开始工作前）

```sh
git fetch upstream
git checkout main && git merge --ff-only upstream/main
git checkout freemagpie && git merge main
# 有冲突时：只保留我们的品牌改动，其余一切以上游为准
git diff main...freemagpie --stat   # 查看我们相对上游的全部改动
```

## 改动原则

1. **改动集中、可枚举**：所有与上游的差异必须登记在下面的「改动清单」里，
   并尽量使用 `// freemagpie:` 注释标记（便于 `git grep -n "freemagpie:"` 审计）。
2. **不碰行为，只换品牌**：网关、路由、协议转换等核心逻辑不改。
   要做功能性定制，先评估它会不会让每次 merge 变难。
3. **文档同步**：本文件在改动范围变化时同一次提交里更新。

## 改动清单（ Keeping this list current ）

> 每一项对应 freemagpie 分支相对 main 的一组改动。
> 同步冲突时，按此清单决定保留谁。

| # | 区域 | 文件 | 内容 | 状态 |
| --- | --- | --- | --- | --- |
| 1 | 品牌标识 | 见 git diff | magpie → FreeMagpie（名称/标题/图标，不含二进制 id `magpie` 与路径 `~/.config/magpie`） | 待做 |
| 2 | 文档 | FREEMAGPIE.md | 本文件 | 已做 |
| 3 | 官网 | site/ | 独立域名站点（暂不改上游 site/，见「官网」一节） | 已做 |
| 4 | 免费积分市场 | plugins/credits/, internal/gui/credits.go, internal/gui/credits_test.go, internal/gui/assets/plugins/credits/, internal/gui/tests/credits-page.test.cjs, internal/gui/api.go（+1 行路由）, plugins/all/all.go（+1 行 import）, internal/gui/assets/index.html, internal/gui/assets/i18n.js, internal/gui/builtins_test.go（内置插件数 + credits 必备）, internal/gui/tests/builtins.test.cjs, internal/settings/settings.go + internal/settingsapi/settingsapi.go（FreeMagpieURL/FreeMagpieQuota 字段）, internal/cli/settings_parity_test.go, internal/gui/settings_keep_test.go | Free Credits Market 内置插件 + 云端目录客户端 | 已做 |
| 5 | 云端客户端 | internal/freemagpie/ | 纯客户端包（catalog/state/cache，不 import internal/gateway），测试齐 | 已做 |
| 6 | Free First 模板 | internal/provider/grouptemplate.go（TemplateFree 分支 + NeedFree）+ grouptemplate_test.go, internal/gui/assets/plugins/routing/client.js, internal/gui/assets/i18n.js, internal/gui/tests/group-template.test.cjs | 一个分组模板（routing=expiring，成员为免费模型：CountFree / Entry.Free / FreeModel），复用 routes.FreeExpiring | 已做 |
| 8 | 独立发布 | .github/workflows/freemagpie-release.yml + freemagpie-complete-release.yml, freemagpie-brand-downloads.yml, build/freemagpie/, internal/update/update.go + notes.go + freemagpie_test.go | 独立构建 13 个品牌安装包及旧名兼容附件、校验和、更新元数据；通过链接器覆盖更新源与发行页，保留上游其他服务地址 | 已做 |
| 9 | 独立数据目录 | Makefile、internal/appdir/freemagpie*、appdir.go、internal/cli、settings/migrate.go、daemon、autostart、GUI GTK 名称、Windows 安装记录、Linux 卸载路径、发布打包脚本 | 配置/缓存/日志/系统数据目录独立化与显式旧配置导入，详见 distribution-storage | 已实现，验证中 |
| 7 | 架构文档 | docs/freemagpie/ | FreeMagpie 架构方案（本分支的产品方向文档） | 已做 |

> 第 4-6 项的设计依据见 docs/freemagpie/ARCHITECTURE.md。merge 冲突时：
> `internal/gui/api.go` 与 `plugins/all/all.go` 各只有一行，冲突取两边；
> `internal/provider/grouptemplate.go` 的 templateOf 尾部追加一个 case，
> 常量/名字表各追加一行，冲突取两边；`internal/gui/assets/i18n.js`
> 的 freemagpie 块是注释标记的独立条目，冲突取两边；其余为新增目录，
> 上游不会碰。
>
> 网站架构（2026-11-13 定）：freemagpie-web 照上游 usemagpie.ai 的
> Cloudflare Workers + 静态 HTML + D1 架构做（不用 Next.js），公共目录在
> CF 边缘、账户数据在 Go 云端，见 ARCHITECTURE.md §5。

## 品牌改动的边界（重要）

改名时**只改面向用户的文案**，除下述已明确变更的配置目录外，以下协议标识保持 `magpie` 不动，否则
会破坏用户数据兼容性和协议标识：

- 配置目录已按用户要求改为 `~/.config/freemagpie/`；`providers.json`、`affinity.json` 等数据文件格式和名称保留
- HTTP 标识：`GET /` 返回的 `{"name":"magpie"}`、`magpie-<agent>` token、
  `magpie://` 链接、`X-Magpie-Session` 头、`MAGPIE_*` 环境变量
- `claude-magpie` 启动器、codex 配置里的 `[model_providers.magpie]`
- 检测代理时 ping 的 `magpieAt` 等行为契约（文档里已声明的对外行为）

理由：升级用户从官方 magpie 迁移过来时数据无缝；上游测试
（4240 个 Go 测试 + 1436 个 JS 测试）大量断言这些字符串，改了会天天冲突。

## 官网（freemagpie.com）

上游 `site/` 是 usemagpie.ai 的静态站（已构建产物 + 文档页）。
两个选择：

- A. 只做跳转/下载页（推荐起步）：单独建一个仓库放 freemagpie.com 的
  站点，主仓库的 `site/` 保持上游原样，merge 零冲突。
- B. 改上游 site/：把品牌文案全换掉，但 site/ 有大量构建产物
  （`site/public/`），每次 merge 都要重新构建，维护成本高。

当前选择：A（主仓库保持 site/ 原样）。

**已落地**（[cq0206/freemagpie-site](https://github.com/cq0206/freemagpie-site)，
本地检出在主仓库旁的 `freemagpie-site/`，已加入 .gitignore）：

- 官网由独立仓库的 Cloudflare Workers 静态资产部署，不需要前端构建。
- 地址：`https://freemagpie.com`、`https://www.freemagpie.com`。
- 域名仍在阿里云续费，DNS 已由 Cloudflare 管理：
  `arely.ns.cloudflare.com`、`brady.ns.cloudflare.com`。
- 官网下载使用本仓库 Releases 的已发布安装包；`/api/latest` 与
  `/api/notes` 提供本发行版的更新元数据，不能指向上游发行版。

## 独立发行

FreeMagpie 的发行分支是 `freemagpie`，使用 **`freemagpie-vX.Y.Z`** 标签，
不要创建 `v*` 标签（会触发上游发布交接工作流）。与上游 code-standards 的
上游发行条件相对应，本发行版要求：标签来自 `freemagpie`；同一次工作流的
全量 nogui 测试、平台构建和原生 smoke 检查通过后，才在
`cq0206/freemagpie` 发布完整资产并标为 Latest。不借用上游的签名服务。

```sh
# 在 freemagpie 分支提交并推送发布配置后：
git tag freemagpie-v0.1.0
git push origin freemagpie-v0.1.0
# 等 FreeMagpie Release 全部通过，确认 Releases 包含 13 个 FreeMagpie 程序文件、13 个旧名兼容程序文件、
# LICENSE、THIRD_PARTY_NOTICES、latest.json、notes.json、SHA256SUMS。
```

`build/freemagpie/package.sh` 使用原有 Makefile，只覆盖版本与更新地址。
`manifest.py` 拒绝缺包、空包，计算真实 SHA-256。各平台原生构建桌面包；
Windows 桌面包交叉编译后在 Windows runner 执行 x64 smoke 检查。
Windows ARM64 与 Android 包只交叉编译，尚未做原生设备启动检查。
首次发行固定在现有 fork 的源代码快照，不把上游同步夹入发布变更。

GitHub macOS ARM64 runner 无容量时，可使用恢复工作流
`Complete FreeMagpie Release`：从同一标签在本机 Mac ARM64 原生构建并 smoke
检查，将四个 Mac ARM64 文件上传到 Draft Release；输入原 Release run ID 与
版本。恢复流程校验原 run 的提交等于标签、全量测试及另外三个平台构建均成功，
重新在 Windows 原生执行 x64 程序，并检查 26 个程序附件、许可、元数据及 SHA-256
齐全后才公开 Release。先取消原 run，避免两个发布者竞争。

macOS 包只有 ad-hoc 签名，未经 Apple 公证；Windows 没有发布者签名。
不得宣传为已公证、免安全提示。官网与发行说明写明安装步骤。
官网下载文件使用 `freemagpie-*`，macOS 包内为 `FreeMagpie.app`，CFBundleName 与显示名均为 FreeMagpie。保留 `magpie-*` 兼容附件供已安装版本更新；内部二进制名与协议保持兼容；配置、缓存、日志和系统应用数据使用独立 freemagpie 命名空间。2026-11-18 对 0.1.0 补发品牌包装，程序代码未变。

官网发布步骤（在 `freemagpie-site` 仓库）：

1. 下载按钮使用 GitHub Releases 的 Latest 安装包链接；Worker 的
   `/api/latest`、`/api/notes` 从同一 Latest 版本读取已校验的 JSON 元数据。
2. `wrangler deploy`，在线验证 HTTPS、更新接口、文件下载及 SHA-256。
3. 之后发布新版本会自动更新下载和更新源；元数据最多缓存 5 分钟。


## 构建

```sh
# Go 工具链（本机装在 ~/sdk/go1.26.8）
export PATH="$HOME/sdk/go1.26.8/bin:$PATH"
export GOFLAGS=-mod=mod
# GOPROXY 已设为七牛镜像（goproxy.cn），如未设：
export GOPROXY=https://goproxy.cn,direct

make build        # 桌面版（macOS 需 Xcode CLT）
make cli          # 纯终端版（交叉编译友好）
go test ./...     # 测试（跑全量很慢，按包跑）
```

注意：Makefile 有自我保护，必须在仓库根目录运行（`make -C <repo>` 可以）。

已发布版本的品牌包装修复可运行 `Complete FreeMagpie download branding`。工作流先验证旧校验和，在原生 Mac runner 重新包装并签名验证两个 FreeMagpie Mac ZIP，直接复制其余平台相同程序为品牌名称，补齐清单与校验和。不会重编译或替换旧名程序。


## 独立数据目录与 Apple 公证

2026-11-18 按用户要求隔离全部应用数据目录，新增 `appdir.Name` 分发构建参数、独立 OS 缓存/日志/WebView/GTK 数据和安装记录。旧数据仅通过显式导入复制，不覆盖已有 FreeMagpie 数据。详细路径、迁移、失败行为与验证见 [distribution-storage](docs/subsystems/distribution-storage.md)。这项用户要求取代原有“共享上游配置目录”的边界。

公证脚本已提供，但当前机器没有 Developer ID Application 证书，尚未向 Apple 提交公证。开发者会员与证书就绪后，在 Keychain 配置 notarytool profile，设置 `FREEMAGPIE_SIGNING_IDENTITY` 和 `FREEMAGPIE_NOTARY_PROFILE` 再运行发布打包脚本。只有签名、公证 Accepted、staple 和 Gatekeeper 验证均成功，才能更新官网为“已公证”。凭据不进入仓库。
