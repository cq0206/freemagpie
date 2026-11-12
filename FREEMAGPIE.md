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

## 品牌改动的边界（重要）

改名时**只改面向用户的文案**，以下内容必须保持 `magpie` 不动，否则
会破坏用户数据兼容性和协议标识：

- 配置目录 `~/.config/magpie/`、`providers.json`、`affinity.json` 等文件名
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

- 单页 `index.html`（无构建步骤、无框架、无重图，~14 KB），GitHub Pages
  从 `main` 分支根目录发布，CNAME 指向 freemagpie.com，已强制开启跳转
- logo 复用上游 `site/public/img/magpie-small.svg`（MIT），加
  `fill="currentColor"` 适配深浅色
- 用户侧还需一步：**到域名注册商给 freemagpie.com 添加 DNS 记录**
  （以仓库 Settings → Pages 显示为准）：
  `@` A 记录 → `185.199.108.153` / `185.199.109.153` / `185.199.110.153` /
  `185.199.111.153`；`www` CNAME → `cq0206.github.io`。
  DNS 生效、证书签发后再到 Settings → Pages 勾选 Enforce HTTPS。

## 构建

```sh
# Go 工具链（本机装在 ~/sdk/go1.26.8）
export PATH="$HOME/sdk/go1.26.8/bin:$PATH"
export GOFLAGS=-mod=mod
# GOPROXY 已设为七牛镜像（goproxy.cn），如未设：
export GOPROXY=https://goproxy.cn,direct

make build        # 桌面版（macOS 需 Xcode CLT）
make nogui        # 纯终端版（交叉编译友好）
go test ./...     # 测试（跑全量很慢，按包跑）
```

注意：Makefile 有自我保护，必须在仓库根目录运行（`make -C <repo>` 可以）。
