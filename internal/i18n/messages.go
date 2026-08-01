package i18n

// msg 是一条消息的中英文。
type msg struct {
	zh string
	en string
}

// messages 是 i18n 消息目录：key -> {zh, en}。与 npm 版 src/i18n/messages.ts 保持同步。
// 约定：所有 user-facing 字符串都走 T()；逻辑层不裸写中文。占位符 {0} {1} … 由 T(key, args...) 按序替换。
var messages = map[string]msg{
	// —— CLI ——
	"cli.desc": {
		zh: "Claude Code API 切换器：在官方账号与第三方 Anthropic 兼容 API 间切换。",
		en: "Claude Code API switcher: switch between the official account and third-party Anthropic-compatible APIs.",
	},
	"cli.arg.name":     {zh: "目标配置名；省略则打开交互菜单", en: "target profile name; omit to open the menu"},
	"cli.opt.session":  {zh: "本次启用：仅当前终端设环境变量并启动 claude（阅后即焚）", en: "session: set env for this terminal only and launch claude"},
	"cli.opt.list":     {zh: "列出所有配置及状态", en: "list all profiles and their state"},
	"cli.opt.storeDir": {zh: "覆盖配置存储目录（测试用，默认 ~/.cc-mini）", en: "override store dir (testing; default ~/.cc-mini)"},
	"cli.opt.defaultScope": {
		zh: "设为默认写到哪：user(持久) / process(不落盘 dry-run，测试用)",
		en: "set-default scope: user (persist) / process (dry-run, testing)",
	},
	"cli.opt.lang":    {zh: "本次界面语言：zh / en", en: "UI language for this run: zh / en"},
	"cli.opt.version": {zh: "显示版本号", en: "show version"},
	"cli.opt.help":    {zh: "显示帮助", en: "show help"},

	// —— 列表 / 状态 ——
	"list.default": {zh: "默认配置：{0}", en: "Default: {0}"},
	"state.login":  {zh: "登录态", en: "Logged in"},
	"state.noKey":  {zh: "密钥未填", en: "No key"},
	"state.apiKey": {zh: "密钥·API_KEY", en: "Key · API_KEY"},
	"state.hasKey": {zh: "密钥已设", en: "Key set"},

	// —— 当前终端环境（只读）——
	"terminal.current":   {zh: "当前终端：{0}", en: "Current terminal: {0}"},
	"terminal.official":  {zh: "未设置 / 官方登录态", en: "not set / official login"},
	"terminal.matched":   {zh: "{0} → {1}", en: "{0} -> {1}"},
	"terminal.unmatched": {zh: "{0}（未匹配配置）", en: "{0} (no matching profile)"},

	// —— 供应商显示名（只影响显示，不改变 providers.json 里的 profile key）——
	"provider.official": {zh: "官方", en: "Official"},
	"provider.deepseek": {zh: "DeepSeek", en: "DeepSeek"},
	"provider.zhipu":    {zh: "智谱GLM", en: "Zhipu GLM"},
	"provider.mimo":     {zh: "小米MiMo", en: "Xiaomi MiMo"},

	// —— 菜单通用 ——
	"menu.prompt": {zh: "输入序号 (q 取消): ", en: "Enter number (q to cancel): "},
	"menu.mainTitle": {
		zh: "CC-X v{0}     默认：{1}",
		en: "CC-X v{0}     Default: {1}",
	},
	"menu.mainHint": {
		zh: "↑↓ 选择 · Enter 进入 · e 编辑 · s 启动 · d 设默认 · Shift+↑↓ 排序 · q 退出",
		en: "↑↓ move · Enter open · e edit · s session · d set-default · Shift+↑↓ reorder · q quit",
	},
	"menu.newProfile": {zh: "新增配置", en: "New profile"},
	"menu.exit":       {zh: "退出", en: "Exit"},
	"menu.default":    {zh: "（默认）", en: "(default)"},
	"menu.comingSoon": {zh: "（该功能下一步实现）", en: "(coming in the next step)"},
	// 更新检查：菜单开关两态 + 有新版时的横幅（{0}=新版本号，{1}=升级命令）
	"menu.updateOff":       {zh: "更新检查：关闭", en: "Update check: off"},
	"menu.updateNotify":    {zh: "更新检查：提醒", en: "Update check: notify"},
	"menu.updateAvailable": {zh: "有新版本 {0} · 升级：{1}", en: "New version {0} · upgrade: {1}"},
	"menu.firstRunHint":    {zh: "首次配置：① 选供应商 → ② 填密钥 → ③ 本次启用", en: "First setup: 1. pick a provider -> 2. paste key -> 3. use this session"},

	// —— 动作菜单 ——
	"action.titlePrefix": {zh: "配置：", en: "Profile: "},
	"action.session": {
		zh: "本次启用    — 仅当前终端，立即启动 Claude（并行多终端推荐）",
		en: "Session    — this terminal only, launches Claude now (great for parallel terminals)",
	},
	"action.setDefault": {
		zh: "设为默认    — 新终端裸敲 claude 默认用它（不影响运行中会话）",
		en: "Set default — used by bare claude in new terminals (running sessions unaffected)",
	},
	"action.check":              {zh: "配置自检    — 只读探测 API 地址与密钥", en: "Check config — read-only probe for API URL and key"},
	"action.edit":               {zh: "编辑", en: "Edit"},
	"action.delete":             {zh: "删除", en: "Delete"},
	"action.back":               {zh: "返回", en: "Back"},
	"action.hint":               {zh: "↑↓ 选择 · Enter 确认 · q 返回", en: "↑↓ move · Enter select · q back"},
	"action.deleteConfirm":      {zh: "确认删除 [{0}]? (y/N): ", en: "Delete [{0}]? (y/N): "},
	"action.deleteOfficialWarn": {zh: "建议保留『官方』。", en: `Keeping "Official" is recommended.`},
	"menu.language":             {zh: "切换到 English", en: "Switch to 中文"},

	// —— 通用占位 ——
	"empty.paren": {zh: "(空)", en: "(empty)"},

	// —— 编辑表单 ——
	"edit.title": {zh: "编辑配置 （↑↓ 选要改的项，Enter 进入；↓到底可选保存/放弃）", en: "Edit profile (↑↓ pick a field, Enter to edit; save/discard at bottom)"},
	"edit.hint": {
		zh: "供应商：选后自动填地址/模型 · 备注随便写 · 回车=不改 · 输入 - =清空",
		en: `Provider: auto-fills url/models · Enter=keep · type "-" to clear`,
	},
	"edit.current":              {zh: "当前：{0}", en: "current: {0}"},
	"edit.inputHint":            {zh: "回车=不改，输入=替换，- =清空", en: `Enter=keep, type=replace, "-"=clear`},
	"edit.field.provider":       {zh: "供应商        ", en: "Provider      "},
	"edit.field.note":           {zh: "备注          ", en: "Note          "},
	"edit.field.base":           {zh: "API 地址      ", en: "API URL       "},
	"edit.field.auth":           {zh: "认证字段      ", en: "Auth field    "},
	"edit.field.key":            {zh: "API 密钥      ", en: "API key       "},
	"edit.field.opus":           {zh: "opus  → 模型  ", en: "opus  → model "},
	"edit.field.sonnet":         {zh: "sonnet→ 模型  ", en: "sonnet→ model "},
	"edit.field.haiku":          {zh: "haiku → 模型  ", en: "haiku → model "},
	"edit.field.fable":          {zh: "fable → 模型  ", en: "fable → model "},
	"edit.field.subagent":       {zh: "子代理 → 模型  ", en: "subagent→ model "},
	"edit.action.models":        {zh: "↻ 获取模型列表（从 API 拉取可选模型）", en: "↻ Fetch model list from API"},
	"edit.field.effort":         {zh: "effort 思考档 ", en: "effort level  "},
	"edit.field.disableTraffic": {zh: "禁用非核心流量  ", en: "Disable traffic "},
	"edit.toggleSecretShow":     {zh: "显示密钥明文（当前隐藏）", en: "Show key in plaintext (now hidden)"},
	"edit.toggleSecretHide":     {zh: "隐藏密钥（当前明文）", en: "Hide key (now shown)"},
	"edit.save":                 {zh: "保存并返回", en: "Save & back"},
	"edit.discard":              {zh: "放弃修改", en: "Discard"},
	"edit.nameEmpty":            {zh: "还没选供应商（或自定义名称），未保存。", en: "No provider/name chosen yet; not saved."},
	"edit.customName":           {zh: "自定义供应商名称（回车=不改）: ", en: "Custom provider name (Enter=keep): "},
	"edit.noteInput":            {zh: "备注（回车=不改，- =清空）: ", en: `Note (Enter=keep, "-" clear): `},

	// —— Picker ——
	"pick.hint":              {zh: "↑↓ 选择 · Enter 确认 · q 不改", en: "↑↓ move · Enter select · q keep"},
	"pick.noChange":          {zh: "不修改", en: "(no change)"},
	"pick.manual":            {zh: "手动输入…", en: "Enter manually…"},
	"pick.provider.title":    {zh: "供应商（当前：{0}）", en: "Provider (current: {0})"},
	"pick.provider.none":     {zh: "(未选)", en: "(none)"},
	"pick.provider.custom":   {zh: "自定义（手动填名字）", en: "Custom (type a name)"},
	"pick.providerUrl.title": {zh: "{0} 有多个 API 地址，选一个", en: "{0} has multiple URLs, pick one"},
	"pick.base.title":        {zh: "API 地址（当前：{0}）", en: "API URL (current: {0})"},
	"pick.base.existing":     {zh: "(已有:{0})", en: "(used:{0})"},
	"pick.base.manualInput":  {zh: "手动输入 API 地址（回车=不改，- =清空）: ", en: `Type API URL (Enter=keep, "-" clear): `},
	"pick.auth.title":        {zh: "认证字段（当前：{0}）", en: "Auth field (current: {0})"},
	"pick.auth.token":        {zh: "AUTH_TOKEN  （Bearer，多数第三方中转）", en: "AUTH_TOKEN  (Bearer, most 3rd-party relays)"},
	"pick.auth.apikey":       {zh: "API_KEY  （x-api-key，官方/少数）", en: "API_KEY  (x-api-key, official/few)"},
	"pick.effort.title":      {zh: "effort 思考档（当前：{0}）", en: "effort level (current: {0})"},
	"pick.effort.empty":      {zh: "留空（不设）", en: "Leave empty"},
	"pick.effort.hint":       {zh: "越往后越深入；auto=模型默认 · q 不改", en: "deeper to the right; auto=model default · q keep"},

	"pick.disableTraffic.title": {zh: "禁用非核心流量（当前：{0}）", en: "Disable nonessential traffic (current: {0})"},
	"pick.disableTraffic.yes":   {zh: "是（值=1，GLM 等要求禁用）", en: "Yes (value=1, required by GLM etc.)"},
	"pick.disableTraffic.no":    {zh: "否（留空，不设置）", en: "No (leave empty)"},

	// —— 获取模型列表 ——
	"models.needBaseKey": {zh: "请先填写 API 地址和 API 密钥。", en: "Fill in API URL and key first."},
	"models.fetching":    {zh: "获取模型列表中…", en: "Fetching models…"},
	"models.fail":        {zh: "获取模型列表失败：{0}", en: "Failed to fetch models: {0}"},
	"models.title":       {zh: "模型列表（↑↓ 选择 · Enter 确认 · q 取消）", en: "Models (↑↓ pick · Enter confirm · q cancel)"},
	"models.hint":        {zh: "[1M] = 该模型支持 1M 上下文（选中后可选后缀）", en: "[1M] = model supports 1M context (suffix pick after select)"},
	"models.ctxTitle":    {zh: "上下文窗口", en: "Context window"},
	"models.ctx1m":       {zh: "1M 上下文（自动加 [1m] 后缀，推荐）", en: "1M context (auto [1m] suffix, recommended)"},
	"models.ctx200k":     {zh: "200K 上下文（无后缀）", en: "200K context (no suffix)"},
	"models.ctxHint":     {zh: "↑↓ 选择 · Enter 确认 · q 取消", en: "↑↓ pick · Enter confirm · q cancel"},
	"models.applyTitle":  {zh: "应用到哪个档位？", en: "Apply to which slot?"},
	"models.applyAll":    {zh: "全部档位（opus/sonnet/haiku/fable）", en: "All slots (opus/sonnet/haiku/fable)"},
	"models.applyHint":   {zh: "↑↓ 选择 · Enter 应用 · q 取消", en: "↑↓ pick · Enter apply · q cancel"},

	// —— 错误 ——
	"error.notFound":     {zh: "找不到配置：{0}", en: "Profile not found: {0}"},
	"error.existing":     {zh: "现有：{0}", en: "Existing: {0}"},
	"error.notFoundHint": {zh: "下一步：运行 `xx -l` 查看配置，或运行 `xx` 打开菜单。", en: "Next: run `xx -l` to list profiles, or `xx` to open the menu."},
	"error.storeRead":    {zh: "配置文件读取失败：{0}", en: "Failed to read config file: {0}"},
	"error.storeCorrupt": {zh: "配置文件解析失败（JSON 语法错误）：{0}", en: "Failed to parse config file (invalid JSON): {0}"},
	"error.storeFormat":  {zh: "配置文件结构不正确（顶层须为对象、providers 须为数组且条目结构合法）：{0}", en: "Config file has invalid structure (top-level must be an object, providers must be an array with valid profile entries): {0}"},
	"error.storeSave":    {zh: "配置文件保存失败：{0}", en: "Failed to save config file: {0}"},
	"error.storeCorruptHint": {
		zh: "为避免误删，未对它做任何改动。请修复后重试；或删除该文件以重新生成默认配置（会丢失已填的密钥）。",
		en: "Left untouched to avoid data loss. Fix it and retry; or delete the file to regenerate defaults (loses any saved keys).",
	},
	"error.storeBackupHint": {zh: "修复前先备份：{0}", en: "Back it up first: {0}"},

	// —— 本次启用（session）——
	"session.noKey": {zh: "⚠ 配置 [{0}] 还没填密钥。", en: "⚠ Profile [{0}] has no key set."},
	"session.launch": {
		zh: "▶ 本次启用：{0}（仅当前终端，不影响其它终端）",
		en: "▶ Session: {0} (this terminal only; others unaffected)",
	},
	"session.starting": {
		zh: "正在启动 Claude…（退出 Claude 后回到命令行）",
		en: "Launching Claude… (returns here after Claude exits)",
	},
	"session.noClaude":     {zh: "未找到 claude 命令，请确认它在 PATH 中。", en: "claude not found on PATH."},
	"session.noClaudeHint": {zh: "下一步：安装 Claude Code 后新开终端；npm 用户可运行 `npm install -g @anthropic-ai/claude-code`。", en: "Next: install Claude Code and open a new terminal; npm users can run `npm install -g @anthropic-ai/claude-code`."},
	"session.onboardingHint": {
		zh: "提示：若 Claude Code 首次启动出现登录引导，选择跳过或关闭引导后第三方 API 会自动生效。",
		en: "Tip: if Claude Code shows a login prompt on first launch, dismiss or skip it — the third-party API will take effect regardless.",
	},

	// —— 配置自检（只读网络探测）——
	"check.noUrl":    {zh: "官方登录态 / 未设置 API 地址，无需自检。", en: "Official login / no API URL set; nothing to probe."},
	"check.noKey":    {zh: "还没填密钥，无法自检。", en: "No key set; cannot probe."},
	"check.ok":       {zh: "✓ 配置自检通过（HTTP {0}）", en: "✓ Config check passed (HTTP {0})"},
	"check.auth":     {zh: "鉴权失败（HTTP {0}），检查密钥和认证字段。", en: "Authentication failed (HTTP {0}); check key and auth field."},
	"check.dns":      {zh: "DNS 解析失败，检查 API 地址或网络。", en: "DNS lookup failed; check API URL or network."},
	"check.timeout":  {zh: "连接超时，请稍后重试或检查网络。", en: "Connection timed out; retry later or check network."},
	"check.notFound": {zh: "接口不存在（HTTP {0}），检查 API 地址是否缺少 /anthropic 等后缀。", en: "Endpoint not found (HTTP {0}); check whether the API URL is missing a suffix like /anthropic."},
	"check.http":     {zh: "请求返回 HTTP {0}，检查 API 地址、密钥或账户状态。", en: "Request returned HTTP {0}; check API URL, key, or account state."},
	"check.network":  {zh: "网络请求失败，检查网络或 curl 是否可用。", en: "Network request failed; check network or curl availability."},

	// —— 设为默认（default）——
	"default.writing": {zh: "正在写入用户环境变量…", en: "Writing user environment variables…"},
	"default.noKey": {
		zh: "⚠ {0} 还没填密钥；设为默认后，新终端的 claude 仍不可用。",
		en: "⚠ {0} has no key; after setting it as default, claude in new terminals will still fail.",
	},
	"default.done": {
		zh: "✓ 已设为默认：{0}  ·  新开终端裸敲 claude 生效（不影响运行中会话）",
		en: "✓ Default set: {0}  ·  effective in newly opened terminals (running sessions unaffected)",
	},
	"default.hintSession": {
		zh: "若只想本次使用：xx -s {0}",
		en: "To use this session only: xx -s {0}",
	},
	"default.dryRun": {
		zh: "（dry-run：--default-scope process，未改系统，仅更新存储）",
		en: "(dry-run: --default-scope process; system untouched, store only)",
	},
	"default.unixWrote": {
		zh: "已写入 {0}（新开终端生效；或 source 它立即生效）",
		en: "Wrote {0} (effective in new terminals; or source it now)",
	},
	"default.failed": {zh: "设为默认失败：{0}", en: "Failed to set default: {0}"},
	"default.fishUnsupported": {
		zh: "⚠ 检测到 fish：v1 暂不支持「设为默认」，请手动设置或改用「本次启用」。",
		en: `⚠ fish detected: "set default" is unsupported in v1; set manually or use session launch.`,
	},
}
