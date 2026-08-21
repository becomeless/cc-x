# 发布流程

ccx 有两条分发线：

- **Go 原生版**（推荐）：GitHub Releases 提供 `xx` / `xx.exe`，覆盖 Windows x64、macOS Intel / Apple Silicon、Linux x64 / arm64，无需 Node.js。
- **npm 版** `@cc-x/cc-x`：面向已有 Node.js 的用户，作为全平台兜底安装线保留。

本文是 Go 原生版的发布流程。命令在仓库根目录执行。

## 前置工具

- Go 工具链（`go` 在 PATH 中；脚本也支持 `-GoExe <path>` 指定）。
- `tar`（打包 `.tar.gz` 资产需要；Windows 10/11 自带）。
- `gh` CLI（创建 GitHub Release）。
- Node.js（仅用于跑 npm 侧的检查）。

## 版本号约定

- Go 原生版与 npm 版共用同一套版本号（当前 `v0.4.x`），后续按 patch 递增。
- 二进制内部版本号在构建时通过 `-ldflags "-X main.version=<version>"` 注入，**不要**复用已发布的 tag。
- `xx --version` 必须打印对应版本号，不能是 `dev`。

## 1. 发布前检查

```powershell
go test ./...
go vet ./...
go build ./cmd/xx
npm run typecheck
npm run build
npm pack --dry-run
```

确认：

- `go test` / `go vet` / `go build` 全绿。
- `npm pack --dry-run` **不含** Go 二进制、Release 压缩包、`providers.json` 或 `node_modules`。
- 真终端 smoke 通过：主菜单、动作菜单、编辑表单、picker、排序、删除、语言切换、`--default-scope process`。
- Windows 实测：`install.ps1`、`xx --version` / `--list` / `xx <name>` / 设为默认。
- macOS / Linux 实测：`install.sh`、`xx --version` / `--list` / `xx <name>`、设为默认写入并可 `source` 对应 rc 文件。

## 2. 构建多平台资产

```powershell
.\scripts\build-release.ps1 -Version 0.4.2
Get-ChildItem .\dist\release
```

产物（`dist\release\`）：

- `ccx_0.4.2_windows_amd64.zip`
- `ccx_0.4.2_darwin_amd64.tar.gz` / `ccx_0.4.2_darwin_arm64.tar.gz`
- `ccx_0.4.2_linux_amd64.tar.gz` / `ccx_0.4.2_linux_arm64.tar.gz`
- `install.ps1`、`install.sh`
- `checksums.txt`（覆盖以上全部资产的 SHA256）

每个压缩包内含 `xx` / `xx.exe`、`presets.json`、`LICENSE`、`README.md`、`README.en.md`。脚本会对当前宿主可运行的那个平台校验 `xx --version` 是否等于注入的版本号。

## 3. 创建 GitHub Release

```powershell
git tag -a v0.4.2 -m "ccx Go native v0.4.2"
git push origin main --tags
gh release create v0.4.2 `
  .\dist\release\ccx_0.4.2_windows_amd64.zip `
  .\dist\release\ccx_0.4.2_darwin_amd64.tar.gz `
  .\dist\release\ccx_0.4.2_darwin_arm64.tar.gz `
  .\dist\release\ccx_0.4.2_linux_amd64.tar.gz `
  .\dist\release\ccx_0.4.2_linux_arm64.tar.gz `
  .\dist\release\install.ps1 `
  .\dist\release\install.sh `
  .\dist\release\checksums.txt `
  --repo becomeless/cc-x `
  --title "ccx v0.4.2" `
  --notes "Go native builds for Windows x64, macOS Intel/Apple Silicon, Linux x64/arm64. npm @cc-x/cc-x remains available."
```

> README 的安装命令依赖 `releases/latest/download/install.ps1` 与 `install.sh`，所以**安装器必须随每个 Release 一起上传**。需要先验下载链路时可加 `--draft` 建草稿，确认后再发布。

## 4. 发布后验证

Windows（新开 PowerShell）：

```powershell
irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1 | iex
xx --version
xx --list
```

macOS / Linux（新开 shell）：

```bash
curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh
xx --version
xx --list
```

确认：

- `xx --version` 为新版本；`xx --list` 能读到现有 `~/.cc-mini/providers.json`。
- Windows：`Get-Command xx` 指向 `%LOCALAPPDATA%\Programs\ccx\xx.exe`（或用户自定义目录）。
- macOS / Linux：`command -v xx` 指向 `~/.local/bin/xx`（或 `CCX_INSTALL_DIR`）。
- `xx <name>` 能继承真实 TTY 拉起 `claude`；切到「官方」只清理 9 个受管环境变量（effort 走 `--effort` 参数，非环境变量），不写任何 Claude Code 配置文件。

## 5. 回滚

```powershell
gh release delete v0.4.2 --repo becomeless/cc-x
git push origin :refs/tags/v0.4.2
git tag -d v0.4.2
```

若已有用户下载到问题版本，优先发下一个 patch 修正并在 Release notes 说明，**不要复用同一 tag**。

## macOS 签名与 quarantine 口径

ccx 的 macOS 分发是终端 CLI tarball，不是 `.app` / `.dmg` / `.pkg` 双击安装：

- 交叉编译的 Go 二进制默认没有 Developer ID 公证。
- `curl | sh` 在终端下载安装，通常不触发浏览器下载附带的 quarantine。
- 仅当将来要提供 `.pkg` / `.dmg` / Homebrew cask 或双击打开的 app 时，才进入 Apple Developer ID 签名 + notarization 流程。
- README 不应宣称“已由 Apple 验证”。

## npm 版发布

npm 版版本号在 `package.json`，已配 `publishConfig.access = public`：

```bash
npm version <patch|minor|major>
npm publish
```

同一版本号不可重发；Go 原生与 npm 两线的 tag 不要互相复用。
