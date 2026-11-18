FreeMagpie's first independent distribution, based on the `freemagpie` branch of magpie (MIT).

- Native desktop builds for macOS, Windows and Linux (Apple silicon/ARM64 and Intel/x64).
- Terminal builds for macOS, Windows, Linux and Android/Termux ARM64.
- Free Credits Market and the Free First routing template from this fork.
- Independent update feed at freemagpie.com, with SHA-256 verification.

### Install

- **macOS 12+:** unzip `freemagpie-darwin-*.zip`, move `FreeMagpie.app` into Applications. This release is ad-hoc signed, **not Apple notarized**. If macOS blocks the first launch, review the download source and allow it in System Settings → Privacy & Security → Open Anyway.
- **Windows:** run `freemagpie-windows-*.exe`. Microsoft Edge WebView2 Runtime is required. The executable has no publisher certificate; Windows may show an unknown-publisher notice.
- **Linux:** install GTK 3 and WebKitGTK 4.1 (Ubuntu: `sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0`), then `chmod +x freemagpie-linux-*` and run the matching file.
- **Terminal:** download the matching `freemagpie-cli-*`, rename it to `magpie` (`magpie.exe` on Windows), and add it to PATH. Run `magpie --help` or `magpie web` to start the browser interface. Terminal builds require no desktop WebView.
- Compare the downloaded file's SHA-256 with `SHA256SUMS` before running it.

The application keeps upstream's `magpie` executable name, protocol and configuration paths. Please quit an existing magpie instance before opening this distribution. No Apple/Microsoft publisher signing certificate is included.

<!-- lang:zh -->
FreeMagpie 首个独立发行版，基于 magpie（MIT）的 `freemagpie` 分支。

- macOS、Windows、Linux 桌面版，支持 ARM64 与 Intel/x64。
- macOS、Windows、Linux 及 Android/Termux ARM64 终端版。
- 包含本分支的免费积分市场和 Free First 路由模板。
- 更新检查使用 freemagpie.com，安装包提供 SHA-256 校验。

### 安装

- **macOS 12+：**解压对应 ZIP，将 `FreeMagpie.app` 拖入“应用程序”。本版使用临时签名，**未经 Apple 公证**。首次启动若被系统拦截，请确认下载来源后，在“系统设置 → 隐私与安全性 → 仍要打开”中允许。
- **Windows：**运行对应架构的 `freemagpie-windows-*.exe`，需要 Microsoft Edge WebView2 Runtime。本版未配置发布者证书，系统可能提示未知发布者。
- **Linux：**先安装 GTK 3 与 WebKitGTK 4.1（Ubuntu：`sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0`），执行 `chmod +x freemagpie-linux-*` 后运行对应文件。
- **终端版：**下载对应 `freemagpie-cli-*`，重命名为 `magpie`（Windows 为 `magpie.exe`）并加入 PATH，运行 `magpie --help`；`magpie web` 可启动浏览器界面，无需桌面 WebView。
- 运行前可将文件的 SHA-256 与 `SHA256SUMS` 对照。

保留上游的 `magpie` 程序名、协议和配置目录。已有 magpie 用户请先退出旧实例。首版尚未配置 Apple / Microsoft 发布者签名证书。

Packaging correction (2026-11-18): website downloads use `freemagpie-*` and macOS archives contain `FreeMagpie.app`. Legacy `magpie-*` files remain for existing updater compatibility. Program code is unchanged.
