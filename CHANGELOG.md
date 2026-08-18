# 更新日志

## v0.4.27 — 2026-08-18

- 修复：手动输入字段（模型名/密钥/子代理）快速输入丢键——Go 版一次 read 带回多个按键时只认第一个字节，导致「保存后少一两个字母」「退格删不干净」；现逐键处理整段字节（TS 版 Node keypress 本就逐键派发，无此问题）
- 修复：退格擦除全角字符只退 1 列，留下半字残影、用户再按退格误删前一个真实字符——退格/删除现按字符显示宽度重绘光标后内容（Go / TypeScript 双版本对齐）
- 新增：手动输入字段支持光标编辑——←/→ 移动光标、光标处插入、退格删光标前字符、Delete 删光标处字符、Home/End 跳行首行尾（含 rxvt 风格 `ESC[7~`/`ESC[8~`），密钥掩码下同样可用（Go / TypeScript 双版本对齐，含 40+ 编辑器单测）
- 修复：ESC 序列被拆到两次 read 边界时切片越界 panic（`ESC[`/`ESC[3`/`ESCO` 结尾的 chunk）——不完整序列整段丢弃防越界（Go 版）
- 修复：组合附加符（如 e + U+0301）宽度口径统一——单字符宽度不再把 0 列钳成 1 列，与整串宽度一致，两版光标移动行为对齐（Go / TypeScript 双版本对齐）
- 调整：编辑表单模型档位行序改为 Fable → Opus → Sonnet → Haiku（Go / TypeScript 双版本对齐）

## v0.4.26 — 2026-08-07

- 新增：6 个供应商预设——Kimi (月之暗面)、MiniMax、阿里百炼、火山方舟、OpenRouter、百度千帆（Go / TypeScript 双版本对齐）
- 修正：gofmt 对齐——Kimi 块字段对齐到 `Models1M` 列、百度千帆块对齐；移除 MiniMax 冗余 `models_api_map`（两条值均等于默认推导 `{base}/v1/models`）

## v0.4.25 — 2026-08-05

- 新增：`xx update` 自更新命令——Go 原生版从 GitHub Releases 下载替换当前二进制、npm 版走 `npm install -g @cc-x/cc-x@latest` 自动升级
- 修正：dev 分支对齐与短路顺序（审查修正）

## v0.4.24 — 2026-08-02

- 修复：fable 档恢复自动附加 `[1m]` 后缀——v0.4.23 将附加限制为 opus/sonnet 档，实测发现第三方映射到 fable 档的模型无 `[1m]` 标记时 Claude Code 按 200K 管理上下文（`[1m]` 是其读取的能力标记，发送前剥离；当前运行时识别 `fable[1m]`）。修正为 opus/sonnet/fable 档命中供应商 `models_1m` 表才附加，haiku 档仍不附加；官方 Fable 5 不命中 1M 表，不受影响（Go / TypeScript 双版本对齐，含 fable 命中/未命中双用例）

## v0.4.23 — 2026-08-02

- 修复：`[1m]` 后缀仅 opus/sonnet 档自动附加——模型列表选择改由档位感知（`applyModelSelection` / `CanAttach1M`），haiku/fable 档不再附加（官方文档仅 `ANTHROPIC_DEFAULT_OPUS_MODEL` / `ANTHROPIC_DEFAULT_SONNET_MODEL` 文档化支持 `[1m]`；Claude Code 内部 quota probe 等路径仍会字面发送带后缀模型名导致 404）（Go / TypeScript 双版本对齐，含 4 档 × 2 状态矩阵单测）
- 修复：模型列表请求只发配置对应的一种认证头——`fetchModels` 增加 auth 参数（AUTH_TOKEN → `Authorization: Bearer`，API_KEY → `x-api-key`），与连通性检查一致，不再双头齐发；补 `anthropic-version` 头；3xx 重定向一律拒绝（跨主机重定向会泄露认证头——Go/undici 的敏感头剥离列表都不含 `x-api-key`）；响应体上限 1 MiB，超限取消并明确报错（对齐 Go 版 LimitReader 资源边界）（Go / TypeScript 双版本对齐，含请求头/重定向/大响应单测）
- 修复：`~/.claude.json` 新建文件权限 0600——TS 版原子写 temp 显式 `mode 0o600`（与 Go `CreateTemp` 对齐）且改为随机临时名 + `flag:'wx'` 独占创建（不复用 PID 重用/崩溃残留的同名文件、不跟随预置符号链接），失败清理先恢复可写再 unlink 且两步独立执行（Windows 上 temp 被 chmod 成只读后直接删除会残留）；删除 POSIX 不可靠的「只读文件报错」测试（rename 取决于父目录权限，root/容器会绕过），补新建 0600 / 保留原权限 / Windows 失败清理测试（Go / TypeScript 双版本对齐）

## v0.4.22 — 2026-08-02

- 修复：原子写失败清理临时文件——`claudecfg.ts` 的 `writeAtomic` 与 `update` 缓存写（Go + TS）在写入或 rename 失败时清理 `.tmp` 残留，与 Go 版 `defer` 清理语义完全对齐，不留孤儿文件（Go / TypeScript 双版本对齐）

## v0.4.21 — 2026-08-02

- 新增：主菜单「一键免登录」——把 `~/.claude.json` 顶层 `hasCompletedOnboarding` 写为 `true`（mimo 接入文档同款官方免登录做法），下次启动 Claude Code 不再弹登录引导。这是 ccx 写 Claude Code 配置文件的**唯一豁免**：字节级最小修改（单遍扫描替换/插入，绝不整文件 JSON 重排），文件不合法 JSON 或顶层非对象时拒绝写入并报错，值已是 `true` 时幂等不写；符号链接解析后写入（Go / TypeScript 双版本对齐，含 30+ 表驱动单测）
- 移除：启动时的被动 onboarding 检测——不再每次会话启动全量读取解析 `~/.claude.json`（大文件实测 +122ms/次）取一个布尔字段；改为用户主动触发（见上）
- 新增（Go 版）：Unix/macOS 下 CLI `xx <名> -s` 改用 `syscall.Exec` 进程替换——常驻进程开销归零、退出码天然透传；Windows 无此机制保持子进程（8 MB / 0% CPU 不变）；菜单内启动两平台均保持子进程以回到菜单
- 优化：菜单「更新检查：提醒」模式下，`update-check.json` 缓存只在菜单打开时读一次，不再每个键击读盘（Go / TypeScript 双版本对齐）

## v0.4.20 — 2026-08-01

- 新增：模型列表菜单显示友好名——API 返回 `display_name` 且与 ID 不同（如 GLM 的 `GLM-5.2` vs `glm-5.2`）时显示「友好名 (实际ID)」，主标签可读、括号里是选中后真正填入的值；无 `display_name` 或与 ID 相同时显示不变（Go / TypeScript 双版本对齐）
- 修复：编辑表单冒号按显示宽度对齐——标签值去掉手写尾部空格（中英混排时宽度算不准，`→` 宽度还随终端字体变化），渲染时按显示宽度（CJK 算 2 列）统一补齐标签列再拼冒号，中英文都自动对齐（Go / TypeScript 双版本对齐）

## v0.4.18 — 2026-08-01

- 修复：1M 支持表匹配大小写不敏感——API 返回 `GLM-5.2`（大写）也能命中表里的 `glm-5.2`，正确标 `[1M]` 并自动附 `[1m]` 后缀（Go / TypeScript 双版本对齐）
- GLM 预设显式配置模型列表端点 `open.bigmodel.cn/api/anthropic/v1/models`（真实 key 实测有效，防止推导路由变化）
- 新增：`findPresetByBase` / `catalogPreset` / `resolveModelsEndpoint` 单测（按 BASE_URL 匹配预设、端点三层解析）

## v0.4.16 — 2026-08-01

- 新增：编辑表单档位行（opus/sonnet/haiku/fable）支持「从模型列表选择」——从供应商 API 拉取实际可用模型（兼容 Anthropic / OpenAI 两种响应格式，防御「HTTP 200 + 业务错误体」），支持 1M 的模型标 `[1M]` 并自动附 `[1m]` 后缀，一步填入；失败明确报错并回退手动输入（Go / TypeScript 双版本对齐）
- 修复：模型列表端点解析——DeepSeek 的 Anthropic 兼容端点未实现 `GET /v1/models`（有效 key 时 404），预设显式指向 OpenAI 风格端点 `api.deepseek.com/models`；MiMo 端点按 BASE_URL 前缀匹配（按量/TokenPlan 各自正确）；预设匹配改为按 BASE_URL 优先（「DeepSeek 2」等改名配置也能命中，端点/1M 表跟着 base 走）；错误信息带实际请求 URL
- 修复：模型操作错误改由表单顶部 Status 提示条展示，不再被下一次菜单清屏吞掉
- 编辑表单「子代理 → 模型」行为空时显示「默认」（官方默认 = 继承主模型）
- `presets.json` 新增可选字段 `models_api` / `models_api_map`（模型列表端点，缺省推导 `{base}/v1/models`）

## v0.4.13 — 2026-08-01

- 新增：模型映射第四档 `fable`（`ANTHROPIC_DEFAULT_FABLE_MODEL`），对齐 Claude Code 官方 4 个模型别名（fable/opus/sonnet/haiku）；编辑表单新增「子代理 → 模型」行（`CLAUDE_CODE_SUBAGENT_MODEL`），受管键 8→10（Go / TypeScript 双版本对齐）
- subagent 模型预设**不预置默认值**——留空即官方默认（子代理继承主会话模型）；想省钱可在配置中自行填 `haiku`（别名，跟随本配置的 haiku 档）或具体模型 ID
- 新增：编辑表单「↻ 获取模型列表」——从供应商 API 拉取可用模型（`{base}/v1/models`，兼容 Anthropic / OpenAI 两种响应格式，防御 GLM 式「HTTP 200 + 业务错误体」），支持 1M 的模型标 `[1M]` 并可一键附 `[1m]` 后缀，选中后应用到 opus/sonnet/haiku/fable 档位；获取失败回退手敲。`presets.json` 新增可选字段 `models_api`（MiMo 的 models 端点是 OpenAI 风格 `/v1/models`，需显式指定）与 `models_1m`（1M 支持表）

## v0.4.12 — 2026-06-15

- 主菜单大幅减负，聚焦「选哪个配置」：删除标题副标题（只留版本 + 默认）、删除「当前终端」状态行；行内去掉 `密钥已设/密钥·API_KEY/登录态` 文字与 host 列，配置「能不能用」改由名字亮/灰区分（缺密钥=名字变灰 + 末尾 `· 密钥未填`），`effort` 保留并锚成最右一列
- 行布局调整为「名字 + 备注 + 状态」三列，备注紧跟名字（同名配置靠它区分），次要信息一律 dim
- 默认配置去掉 `●` 符号（与光标 `▶` 重叠冗余），改为仅名字加粗标识；底部「切换语言 / 更新检查」两项弱化为 dim
- `xx --list` 同步上述行样式（保留「当前终端」诊断行）；Go / TypeScript 双版本对齐

## v0.4.11 — 2026-06-15

- 美化主菜单与 `xx --list`：改为定宽分栏，配置名为主信息、状态/备注/host 一律弱化（dim）并各自对齐成竖列；去掉 `[]` 括号与 `—` 破折号，分隔符收敛为单个 `·`；默认配置改用加粗 `●` 标记。状态/备注列宽按当前配置内容动态计算，自动适配任意语言与文案。TUI 菜单与 `--list` 共用同一套格式化逻辑（Go / TypeScript 双版本对齐）

## v0.4.10 — 2026-06-14

- 移除 `CLAUDE_CODE_AUTO_COMPACT_WINDOW` 受管键（Claude Code 已通过模型名 `[1m]` 后缀自动感知上下文窗口大小，无需额外设置）；受管键降至 8 个
- 「禁用非核心流量」字段改为是/否 picker，不再需要手动输入 `1`；GLM 等要求禁用的供应商可直接选「是」

## v0.4.9 — 2026-06-14

- 新增：受管环境变量从 7 个扩展至 9 个，新增 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` 和 `CLAUDE_CODE_AUTO_COMPACT_WINDOW`；切换配置时自动清除，编辑表单中可直接填写（Go / TypeScript 双版本对齐）
- 新增：GLM 预设自动预填 `DISABLE_NONESSENTIAL_TRAFFIC=1`（GLM 要求）和 `AUTO_COMPACT_WINDOW=1000000`（匹配 GLM 1M 上下文窗口），通过 `presets.json` → Preset.Env → 编辑表单 picker 传递，新建或重新选择 GLM 配置时自动生效
- 新增（Go 版）：`internal/claudecfg` 包——只读检测 `~/.claude.json` 的 `hasCompletedOnboarding` 字段；非官方配置「本次启用」时若 onboarding 未完成，打印非阻塞提示（该机制已在 v0.4.21 移除，改为主菜单「一键免登录」主动写入）
- 新增：`Preset.Env` 字段（Go + TypeScript），基于白名单规范化，仅允许 CC-behavior 变量（不含鉴权 key）

## v0.4.8 — 2026-06-10

- 修复：Windows 终端将 Enter 以 `\r\n` 分两次投递时，`\n` 残留在 stdin 缓冲区，导致编辑含 `[` 字符的模型名（如 `mimo-v2.5[1m]`）后保存无效——ReadValue 进入 raw 模式后立即将残留 `\n` 误读为「空回车=不改」，用户输入被静默丢弃
- 新增：编辑的配置本身是当前默认配置时，保存成功后自动将新值写入用户环境变量，不再需要手动再执行一次「设为默认」
- 新增：`providers.json` 写盘失败时在菜单黄字横幅提示，不再静默丢弃
- 内部重构：拆分 `PersistEnv`（只写平台环境变量）与 `SetDefault`（写环境变量 + 更新 current + 存盘），避免「编辑后自动同步」路径的重复写盘；Go / TypeScript 双版本对齐
- 修复（安装器）：`xx -s` 会话运行期间执行升级不再报"文件被占用"——新二进制先暂存为 `xx.exe.new`，成功后将旧文件改名为 `xx.exe.<guid>.old`（Windows 允许对运行中文件改名）再将暂存文件改名到位；旧文件在下次升级或卸载时自动清理
- 修复（安装器）：卸载时若有仍被占用的备份文件无法删除，改为打印 warning 并说明手动清理路径，不再静默跳过

## v0.4.7 — 2026-06-06

- 新增「配置自检」：动作菜单一键探测 `{base}/v1/models`，区分 鉴权失败 / DNS 解析失败 / 超时 / 404 / 其他状态，切换前即知配置是否可用（纯只读探测，不写任何文件）
- 菜单顶部 / `--list` 显示「当前终端实际生效的 API」，标题显示当前默认配置，消除双激活模式下「这个终端在用哪家」的困惑
- 主菜单单键快捷键：`e` 编辑 · `s` 本次启用 · `d` 设为默认（无需先进动作菜单）
- 无密钥的第三方配置按 Enter 直达密钥编辑；当所有第三方配置都没填密钥时显示首次配置引导横幅，铺平首次成功路径
- 删除确认改为单键 `y/N`（raw 模式，无需回车），与菜单体验一致
- 英文模式下供应商显示名本地化（Zhipu GLM / Xiaomi MiMo），CLI 支持用英文显示名作为别名匹配
- 错误提示补充可复制的下一步命令（找不到配置 / 未找到 claude / 配置文件损坏的备份命令）
- 缺密钥时「设为默认」给出醒目黄字警告，避免误以为已可用
- 菜单行尾以灰字显示 API host，便于区分同名配置
- 修复：非交互（管道）模式下菜单序号跳号；超宽彩色行截断时的颜色泄漏

## v0.4.6 — 2026-06-06

- 修正 0.4.5 发布标签未包含 README banner 与 CHANGELOG 的问题；重新发布包含最新文档的多平台资产

## v0.4.5 — 2026-06-06

- 二进制体积减少 55%（xx.exe 6.8 MB → 3.1 MB，Windows zip 2.9 MB → 1.3 MB）：将更新检查的网络请求改为 `curl` 子进程，消除 Go TLS/加密栈（80+ crypto/net 包）的引入；编译包数从 208 降至 101

## v0.4.4 — 2026-06-04

- 新增版本更新检查（在主菜单切换开/关）：结果缓存 24 小时，后台异步刷新，不阻塞启动；新版本「下次打开」才提示，不打扰当前使用
- 添加 Go/npm 双版本 i18n 键对拍测试，防止两线文案漂移

## v0.4.3 — 2026-06-04

- 菜单移除行号和图标，界面更简洁
- 新增 RELEASING.md，记录多平台发布流程
- Go 版与 npm 版版本号统一对齐

## v0.4.2 — 2026-06-04

- 发布 macOS（Intel / Apple Silicon）和 Linux（amd64 / arm64）原生二进制，Go 原生版实现全平台覆盖

## v0.4.1 — 2026-06-03

- 修复 Windows 安装脚本误调 GitHub API 导致的限速问题

## v0.4.0 — 2026-06-03

- 发布 Go 原生 Windows（amd64）版：单文件、无需 Node.js、零依赖安装

## v0.3.0 — 2026-06-02

- npm 版（`@cc-x/cc-x`）首发，全平台（Windows / macOS / Linux）覆盖
- 完整 TypeScript 重写：自绘 ANSI TUI（无 inquirer/Ink）、中英双语 i18n、供应商目录驱动的配置向导
- 两种启用模式：「本次启用」（进程级，多终端并行互不干扰）和「设为默认」（用户级，新开终端生效）
- 供应商选择器自动填充 URL / 模型映射 / 鉴权字段；重名配置自动追加编号

## v0.2.3 — 2026-06-01

*PowerShell 版*

- 三级菜单分级返回，记住上次选中项
- 修复「本次启用」启动 claude 时报「Input must be provided」的错误

## v0.2.0 — 2026-06-01

*PowerShell 版*

- 编辑界面改为方向键导航，支持 Esc 取消；中文列按显示宽度对齐；菜单重绘去闪烁
- 「设为默认」直写注册表，大幅提速并显示进度提示
- 供应商目录升级：配置支持多份、可就地拖拽排序
- 主菜单标题加版本号显示

## v0.1.0 — 2026-05-31

*PowerShell 版*

- 初始发布：ccx API 切换器核心功能，两种启用模式，`presets.json` 供应商目录
