<h1 align="center">CC-X</h1>

<p align="center">
  <strong>No config files · Process isolation · Parallel terminals · Zero deps</strong>
</p>

<p align="center">
  <a href="https://github.com/becomeless/cc-x/releases/latest"><img src="https://img.shields.io/github/v/release/becomeless/cc-x?style=flat-square&color=blue" alt="version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/becomeless/cc-x?style=flat-square" alt="license"></a>
  <a href="https://github.com/becomeless/cc-x/releases/latest"><img src="https://img.shields.io/github/downloads/becomeless/cc-x/total?style=flat-square&color=success" alt="downloads"></a>
  <a href="https://github.com/becomeless/cc-x/actions"><img src="https://img.shields.io/github/actions/workflow/status/becomeless/cc-x/release.yml?style=flat-square&label=build" alt="build"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26-blue?style=flat-square&logo=go" alt="go"></a>
  <a href="https://www.npmjs.com/package/@cc-x/cc-x"><img src="https://img.shields.io/npm/v/@cc-x/cc-x?style=flat-square&color=success" alt="npm"></a>
</p>

<p align="center">
  <a href="README.md">🇨🇳 中文</a> · <a href="README.en.md">🇺🇸 English</a>
</p>

<p align="center">
  <a href="#features">Features</a> · <a href="#install">Install</a> · <a href="#60-second-quick-start">Quick Start</a> · <a href="#two-modes-the-key-concept">Concepts</a> · <a href="#configuration">Config</a> · <a href="#project-structure">Structure</a> · <a href="#faq">FAQ</a>
</p>

---

> `xx` — one command to switch Claude Code between APIs. **No API-config clobbering.**

Switching Claude Code between the official account and third-party APIs means juggling
environment variables — or trusting a tool that rewrites your Claude config. CC-X keeps API
endpoints, credentials, and model mappings purely at the environment-variable layer. It never
rewrites your MCP, plugins, hooks, or other Claude Code behavior settings.

```text
  CC-X v0.4.24     Default: Official

     Official
   ▶ DeepSeek     work   · effort=max
     Zhipu GLM           · No key
     Xiaomi MiMo         · No key

     New profile  ·  Switch to 中文  ·  Update check: off  ·  Exit

  ↑↓ move · Enter open · e edit · s session · d set-default · Shift+↑↓ reorder · q quit
```

> **Two builds**: the **native Go build** is recommended — GitHub Releases provide a lightweight
> `xx` / `xx.exe` with no Node.js, for Windows x64, macOS Intel / Apple Silicon, Linux x64 / arm64.
> If you prefer npm, install `@cc-x/cc-x` (command is still `xx`). Both builds are feature-equal.

---

## ✨ Features

- **One job**: switch which API Claude Code uses — clear boundaries, no "control panel on top"
- **Never touches config files**: endpoints, keys, and model mappings live only in environment variables; MCP, plugins, and hooks are untouched
- **Two scopes**: "Use this session" is process-local; "Set default" writes user env vars — parallel terminals never interfere
- **9 pre-seeded providers**: Official + DeepSeek / Zhipu GLM / Xiaomi MiMo / Kimi / MiniMax / OpenRouter / Baidu Qianfan / Alibaba Bailian / Volc Ark — paste a key and go
- **Four-tier model mapping**: opus / sonnet / haiku / fable translate to each provider's real model names, with optional live model-list pull
- **`xx update` self-update**: check, download, replace in place
- **Cross-platform, zero deps**: native Go builds for Windows / macOS / Linux, plus npm (`@cc-x/cc-x`)

---

## 🚀 Install

> Install [Claude Code](https://claude.ai/code) first (`claude` on PATH). **Open a new terminal** after installing.

### Step 1 · Install CC-X

**Windows (native, recommended)**

```powershell
irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1 | iex
```

The installer chooses a per-user directory and adds it to your user PATH automatically, so no administrator rights or manual PATH edits are needed.

**macOS / Linux (native, recommended)**

```bash
curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh
```

The installer places `xx` in a user-level command directory. If that directory isn't on PATH,
it prints a hint (the Unix installer deliberately doesn't edit your shell config).

**npm (any platform, Node.js ≥ 18)**

```bash
npm install -g @cc-x/cc-x
```

### Step 2 · Configure your API key

```bash
xx   # First run seeds 9 presets — pick one, edit, paste your key
```

### Step 3 · Start using it

```bash
xx DeepSeek -s     # Use this session, launch Claude now
xx DeepSeek        # Set as default for new terminals
```

### Updating

**`xx update` does it all** — it checks for the latest release, downloads it, and replaces
the binary in place (the npm edition runs `npm` for you). No install command to remember:

```bash
xx update     # check and update to the latest version
xx --version  # confirm the version
```

> Self-update only replaces the program itself; your config and keys (`~/.cc-mini/`) are untouched.

If self-update is unavailable (old version, read-only install directory, platform outside the
release matrix, …), fall back to **re-running the install command** — the installer downloads
the latest release and overwrites the old binary in place, no uninstall needed. **Open a new
terminal** afterward; `xx --version` should show the new version.

- **Windows**: `irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1 | iex`
- **macOS / Linux**: `curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh`
- **npm**: `npm i -g @cc-x/cc-x@latest`

> With the menu's "Update check" set to "notify", CC-X shows a banner at the top of the menu
> when a new version is out — just run `xx update` (older versions get the install command).

---

## ⏱ 60-second quick start

The first run of `xx` seeds 9 profiles in `~/.cc-mini/providers.json` (Official + 8 third-party), **with empty keys**.

1. `xx` → ↑↓ to a profile → Enter → **Edit** → **API key** → paste your key
2. Then either:
   - **Use this session** — launch Claude now in this terminal (temporary, parallel-friendly)
   - **Set default** — bare `claude` in new terminals uses it from now on

```bash
xx                 # open the menu
xx DeepSeek        # set as default
xx DeepSeek -s     # use this session, launch Claude now (--session)
xx -l              # list all profiles and state (--list)
xx update          # check and update to the latest version
xx --help          # all options
```

---

## 🧭 Two modes (the key concept)

Which API Claude uses is decided by **environment variables**. CC-X offers two scopes:

| | Use this session (`-s`) | Set default |
|---|---|---|
| Mechanism | Sets env vars on this process + launches `claude` | Writes **user environment variables** |
| Scope | This terminal only; **gone when you close it** | **New** terminals going forward |
| Running sessions | Unaffected | Unaffected (env freezes at process start) |
| Best for | Parallel terminals on different APIs | Set your daily-driver API once |

> 💡 **Analogy**: "Use this session" is a quick oil change — just for this trip. "Set default" is
> refilling the tank — every new drive uses it from now on.

**Parallel example**: open 4 terminals and run `xx Official -s`, `xx DeepSeek -s`, `xx "Zhipu GLM" -s`,
`xx "Xiaomi MiMo" -s` — four Claudes running at once, each on its own API, zero interference.

**Why not a global config file?** `settings.json` is shared globally; editing it hits running
sessions (classic symptom: another terminal suddenly says `cannot be parsed as a URL`).
Environment variables are naturally process-isolated.

---

## 🚫 When CC-X is NOT the right tool

- You need to manage MCP, hooks, plugins, or multiple CLIs → use [cc-switch](https://github.com/farion1231/cc-switch)
- You only use the official API, never switch → you don't need CC-X
- You want automatic config migration/backup → that's outside CC-X's scope

CC-X cares more about boundaries than features. It does one thing: **switch APIs**.

---

## ⚖️ CC-X vs cc-switch

cc-switch is an excellent full-featured GUI; CC-X takes the opposite, minimal approach.

| | CC-X (`xx`) | cc-switch |
|---|---|---|
| Form | Terminal command (lightweight) | Desktop GUI (full-featured) |
| Scope | Just API switching | API + MCP + multiple CLIs + prompts… |
| API/model configuration | **Never written to config files** (env vars only) | Written through its own config system |
| Claude config boundary | No MCP/plugin/hook management; only one user-initiated onboarding boolean edit | Broader configuration scope |
| Parallel terminals | **Native** (process isolation) | Global switch; sessions can clash |

- → **CC-X**: terminal natives, parallel-session runners, anyone burned by a config-wrecking switcher, "just switch the API" people
- → **cc-switch**: GUI preference, all-in-one MCP + multi-CLI management

---

## 🧠 Design philosophy

> CC-X cares more about boundaries than features.

Claude Code already has its own config system, MCP ecosystem, and session state. CC-X is not trying to become a control panel above it, or to copy user config into another database. It stands at one narrow point before Claude Code starts: prepare the 9 managed environment variables, then let Claude Code run.

That constraint is deliberate: API endpoints, credentials, and model mappings never go into Claude Code config files; there is no MCP/plugin/hook management, automatic migration, or resident background controller. The only Claude Code config mutation is the explicit "Skip login" action described below, limited to one onboarding boolean. Doing less keeps the failure surface small.

Issues / PRs are welcome, but the direction is clear: **make switching steadier, clearer, and less intrusive** before adding broader management power. The onboarding exception must never be broadened to API/model settings, MCP, plugins, hooks, or other behavior configuration.

---

## ⚙️ Configuration

### Fields

| Field | Environment variable | Notes |
|---|---|---|
| API URL | `ANTHROPIC_BASE_URL` | Third-party endpoint; empty for Official = logged-in session |
| Auth field | — | `AUTH_TOKEN` (default) or `API_KEY`; **wrong one = 401** |
| API key | `ANTHROPIC_AUTH_TOKEN` or `ANTHROPIC_API_KEY` | Value for the chosen auth field |
| opus → model | `ANTHROPIC_DEFAULT_OPUS_MODEL` | Four-tier model mapping; haiku also covers background tasks — **must be set** |
| sonnet → model | `ANTHROPIC_DEFAULT_SONNET_MODEL` | |
| haiku → model | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | |
| fable → model | `ANTHROPIC_DEFAULT_FABLE_MODEL` | Top tier (e.g. Fable 5); leave empty if the provider has none |
| subagent → model | `CLAUDE_CODE_SUBAGENT_MODEL` | Model for subagents/agent teams; **empty = do not force an override; inherit the main model when no other model is specified**. To save cost, fill `haiku` (alias, follows this profile's haiku tier) or a concrete model ID |
| effort level | launch flag `--effort` (stored under key `CLAUDE_CODE_EFFORT_LEVEL`) | `low`–`max`; `auto` = model default; empty = unset. **Acts as the launch default only — `/effort` can switch freely in-session** (officially this env var outranks `/effort`, so it is not injected as a variable, to avoid locking the session). Third parties may not honor it |
| Disable nonessential traffic | `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` disables nonessential Claude Code traffic; empty = unset. Zhipu GLM defaults to `1` |

> CC-X **deliberately does not set** `ANTHROPIC_MODEL`. Use `/model opus|sonnet|haiku|fable` in-session;
> the mapping table translates to the provider's real model name.

### Auth field: AUTH_TOKEN vs API_KEY

| Option | Request header | Used by |
|---|---|---|
| `AUTH_TOKEN` (default) | `Authorization: Bearer <key>` | Most third-party relays |
| `API_KEY` | `x-api-key: <key>` | The official API, and a few relays |

### Pre-seeded profiles

| Profile | BASE_URL | OPUS / SONNET / FABLE | HAIKU | effort | Extra env |
|---|---|---|---|---|---|
| Official | empty (logged-in) | — | — | — | — |
| DeepSeek | `https://api.deepseek.com/anthropic` | `deepseek-v4-pro` | `deepseek-v4-flash` | `max` (recommended) | — |
| Zhipu GLM | `https://open.bigmodel.cn/api/anthropic` | `GLM-4.7` | `glm-4.5-air` | — | `DISABLE_NONESSENTIAL_TRAFFIC=1` |
| Xiaomi MiMo | `https://api.xiaomimimo.com/anthropic` | `mimo-v2.5-pro` | `mimo-v2.5-pro` | — | — |
| Kimi | `https://api.moonshot.cn/anthropic` | `kimi-k3` | `kimi-k3` | `max` (recommended) | — |
| MiniMax | `https://api.minimaxi.com/anthropic` | `MiniMax-M3` | `MiniMax-M3` | — | — |
| OpenRouter | `https://openrouter.ai/api` | `anthropic/claude-sonnet-5` | `anthropic/claude-haiku-4-5` | — | — |
| Baidu Qianfan | `https://qianfan.baidubce.com/anthropic/coding` | `qianfan-code-latest` | `qianfan-code-latest` | — | `DISABLE_NONESSENTIAL_TRAFFIC=1` |
| Alibaba Bailian | `https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic` | `qwen3.8-max` | `qwen3.6-flash` | — | — |
| Volc Ark | `https://ark.cn-beijing.volces.com/api/coding` | `doubao-seed-2.0-code` | `doubao-seed-2.0-code` | — | — |

> Model names change as providers update. Several providers offer multiple endpoints
> (e.g. pay-as-you-go vs Token Plan); you pick one when selecting the provider.
> Editing any model tier (opus/sonnet/haiku/fable) offers "Pick from model list" — pulls the
> provider's actual models; models supporting 1M context (e.g. `deepseek-v4-pro`, `glm-5.2`,
> `mimo-v2.5-pro`) are marked `[1M]` and get the `[1m]` suffix automatically — **on the
> opus/sonnet/fable tiers** (Claude Code reads the `[1m]` marker to enable extended context; a
> third-party Fable mapping without it is managed at 200K). The haiku tier never appends the suffix.
> Falls back to manual input on failure.

### Advanced

- **Multiple accounts**: create multiple profiles from the same provider — names auto-suffix
  with ` 2`, ` 3`… Use **Note** to tell them apart, shown as "Provider — Note".
- **Custom providers**: `presets.json` is the provider catalog; add a JSON entry to offer a new
  one, no code change. Drop `~/.cc-mini/presets.json` to override the shipped catalog.
- **First-launch login prompt**: the "Skip login" entry in the main menu writes
  `hasCompletedOnboarding: true` in `~/.claude.json` — the officially recommended way to bypass
  the login wizard (same approach as the MiMo integration guide). The next Claude Code launch
  skips onboarding. This is the single exception to CC-X's no-write rule: only that one
  top-level boolean field is touched, byte-level, and an invalid file is refused untouched.
- **Update check**: toggle to "notify" in the menu — a yellow one-liner appears atop the menu
  when a new release is out. At most one check per day; never auto-upgrades.

---

## 💾 Data & files

- **Profiles (plaintext keys, keep local)**: `~/.cc-mini/providers.json` (also holds `lang` and `update`)
- **Provider catalog**: shipped `presets.json`; override at `~/.cc-mini/presets.json`
- **"Set default" writes user environment variables** (not Claude config files):
  - Windows → registry `HKCU\Environment` + one change broadcast
  - Unix → `# >>> xx >>>` … `# <<< xx <<<` marker block in shell startup file (idempotent rewrite, chosen by `$SHELL`)
  - Same semantics either way: **only affects new terminals**; switching to "Official" clears all managed vars
- **Claude Code behavior configuration is not managed or rewritten.** The sole config mutation is
  the explicit, user-initiated "Skip login" action, which writes only the top-level
  `hasCompletedOnboarding` boolean in `~/.claude.json` (byte-level minimal atomic edit; invalid JSON
  is refused). It never writes API/model settings, MCP, plugins, or hooks.

CC-X only touches these 9 "managed" variables (and clears the ones a target profile doesn't use):
`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_API_KEY`, `ANTHROPIC_DEFAULT_OPUS_MODEL`,
`ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `ANTHROPIC_DEFAULT_FABLE_MODEL`,
`CLAUDE_CODE_SUBAGENT_MODEL`, `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`.
The effort level is the one exception: it is still stored in the profile's env table (key `CLAUDE_CODE_EFFORT_LEVEL`),
but injected as the `--effort` launch flag rather than an environment variable — the official docs state this
variable outranks `/effort`, and injecting it would lock in-session switching (see the table above).

> 💡 To change `settings.json`, use Claude Code's own `/update-config` and describe what you want
> in natural language (e.g. "allow npm commands") — safer than letting an external tool rewrite it.

---

## ❓ FAQ

**Does switching in one terminal affect another?** No. "Use this session" is process-scoped;
"Set default" only affects new terminals.

**I set default but bare `claude` here is still the old one?** Expected — this terminal has
the old env. Open a new one.

**Seeing `cannot be parsed as a URL`?** A profile's API URL is invalid. Edit to fix or delete it.

**Set effort on a third party but nothing happens?** effort is a Claude-model feature; third
parties may not support it. DeepSeek recommends `max`; leave empty otherwise.

**Are keys safe?** Plaintext in your home dir, protected by your OS account. Don't commit
`providers.json` to a repo.

**Can I choose the install directory?** Yes. The Windows installer supports `-InstallDir`;
macOS / Linux supports `CCX_INSTALL_DIR` or `--install-dir`. Most users should keep the default;
if you change it, pass the same directory when uninstalling.

**Can I download the binary manually?** Yes. Go to [GitHub Releases](https://github.com/becomeless/cc-x/releases/latest),
download the zip / tar.gz for your platform, extract it, and put `xx` / `xx.exe` somewhere on PATH.
For most users, the install command above is better: it picks the platform, verifies checksums, and handles PATH / uninstall.

---

## 🗑 Uninstall

1. Clear env vars: `xx` → Official → Set default
2. Remove the binary:
   - Windows native:
     ```powershell
     $s = irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1
     & ([scriptblock]::Create($s)) -Uninstall
     ```
     This removes the installed files and automatically removes the matching user PATH entry.
   - macOS / Linux native:
     ```bash
     curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh -s -- --uninstall
     ```
   - npm: `npm uninstall -g @cc-x/cc-x`
3. Delete data: `rm -rf ~/.cc-mini`

---

## 📁 Project structure

```
cmd/               Go CLI entrypoints (xx, tui-probe)
internal/          Core implementation: config, env vars, platform adapters, TUI, self-update
src/               npm build source (TypeScript)
scripts/           Build/release scripts
docs/screenshots/  UI screenshots
presets.json       Provider catalog (overridable at ~/.cc-mini/presets.json)
install.ps1 / install.sh    Platform installers
CHANGELOG.md       Changelog
```

---

## 📄 License

[MIT](LICENSE)
