<h1 align="center">CC-X</h1>

<p align="center">
  <strong>No config files · Process isolation · Parallel terminals · Zero deps</strong>
</p>

<p align="center">
  <a href="https://github.com/becomeless/cc-x/releases/latest"><img src="https://img.shields.io/github/v/release/becomeless/cc-x?style=flat-square&color=blue" alt="version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/becomeless/cc-x?style=flat-square" alt="license"></a>
  <a href="https://github.com/becomeless/cc-x/releases/latest"><img src="https://img.shields.io/github/downloads/becomeless/cc-x/total?style=flat-square&color=success" alt="downloads"></a>
  <a href="https://github.com/becomeless/cc-x/actions"><img src="https://img.shields.io/github/actions/workflow/status/becomeless/cc-x/release.yml?style=flat-square&label=build" alt="build"></a>
</p>

<p align="center">
  <a href="README.md">🇨🇳 中文</a> · <a href="README.en.md">🇺🇸 English</a>
</p>

<p align="center">
  <a href="#-features">Features</a> · <a href="#-install">Install</a> · <a href="#-60-second-quick-start">Quick Start</a> · <a href="#-two-modes-the-key-concept">Concepts</a> · <a href="#-configuration">Config</a> · <a href="#-faq">FAQ</a>
</p>

---

> `xx` — one command to switch Claude Code between APIs. **Zero config risk.**

Switching Claude Code between the official account and third-party APIs means juggling
environment variables — or trusting a tool that rewrites your Claude config. CC-X takes a
different path: **switching happens purely at the environment-variable layer.** It never
reads or writes any Claude Code config file. Your MCP, plugins, hooks — it won't touch them.

```text
  CC-X v0.4.8 · Claude Code API switcher     Default: Official

   ▶ Official          (default)[Logged in]
     DeepSeek            [Key set] — work
     Zhipu GLM           [No key]
     Xiaomi MiMo         [No key]

     New profile  ·  Switch to 中文  ·  Update check: off  ·  Exit

  ↑↓ move · Enter open · e edit · s session · d set-default · Shift+↑↓ reorder · q quit
```

> [!NOTE]
> **Two builds**: the **native Go build** is recommended — GitHub Releases provide a lightweight
> `xx` / `xx.exe` with no Node.js, for Windows x64, macOS Intel / Apple Silicon, Linux x64 / arm64.
> If you prefer npm, install `@cc-x/cc-x` (command is still `xx`). Both builds are feature-equal.

---

## ✨ Features

- **🛡️ No config files** — switching lives entirely at the env-var layer; it never reads or writes any Claude Code config file. MCP, plugins, hooks stay untouched.
- **🧩 Process isolation** — each terminal sets its own env vars, so they never interfere, sidestepping the "edit a global file and break a running session" trap.
- **⚡ Parallel terminals** — open many terminals, each on its own API; multiple Claudes run at once without clashing.
- **📦 Zero deps** — a single native Go binary, no Node.js, for Windows / macOS / Linux alike.

---

## 📦 Install

> [!IMPORTANT]
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
xx   # First run seeds 4 presets — pick one, edit, paste your key
```

### Step 3 · Start using it

```bash
xx DeepSeek -s     # Use this session, launch Claude now
xx DeepSeek        # Set as default for new terminals
```

### Updating

Updating just means **re-running the install command** — the installer downloads the latest
release and overwrites the old binary in place, no uninstall needed. **Open a new terminal**
afterward; `xx --version` should show the new version.

- **Windows**: `irm https://github.com/becomeless/cc-x/releases/latest/download/install.ps1 | iex`
- **macOS / Linux**: `curl -fsSL https://github.com/becomeless/cc-x/releases/latest/download/install.sh | sh`
- **npm**: `npm i -g @cc-x/cc-x@latest`

> [!TIP]
> With the menu's "Update check" set to "notify", CC-X shows a banner at the top of the menu
> when a new version is out, with the matching upgrade command for your platform.

---

## 🚀 60-second quick start

The first run of `xx` seeds 4 profiles in `~/.cc-mini/providers.json` (Official + DeepSeek +
Zhipu GLM + Xiaomi MiMo), **with empty keys**.

1. `xx` → ↑↓ to a profile → Enter → **Edit** → **API key** → paste your key
2. Then either:
   - **Use this session** — launch Claude now in this terminal (temporary, parallel-friendly)
   - **Set default** — bare `claude` in new terminals uses it from now on

```bash
xx                 # open the menu
xx DeepSeek        # set as default
xx DeepSeek -s     # use this session, launch Claude now (--session)
xx -l              # list all profiles and state (--list)
xx --help          # all options
```

---

## 🎯 Two modes (the key concept)

Which API Claude uses is decided by **environment variables**. CC-X offers two scopes:

| | Use this session (`-s`) | Set default |
|---|---|---|
| Mechanism | Sets env vars on this process + launches `claude` | Writes **user environment variables** |
| Scope | This terminal only; **gone when you close it** | **New** terminals going forward |
| Running sessions | Unaffected | Unaffected (env freezes at process start) |
| Best for | Parallel terminals on different APIs | Set your daily-driver API once |

> [!TIP]
> **Analogy**: "Use this session" is a quick oil change — just for this trip. "Set default" is
> refilling the tank — every new drive uses it from now on.

**Parallel example**: open 4 terminals and run `xx Official -s`, `xx DeepSeek -s`, `xx "Zhipu GLM" -s`,
`xx "Xiaomi MiMo" -s` — four Claudes running at once, each on its own API, zero interference.

**Why not a global config file?** `settings.json` is shared globally; editing it hits running
sessions (classic symptom: another terminal suddenly says `cannot be parsed as a URL`).
Environment variables are naturally process-isolated.

---

## ⚙️ Configuration

### Fields

| Field | Environment variable | Notes |
|---|---|---|
| API URL | `ANTHROPIC_BASE_URL` | Third-party endpoint; empty for Official = logged-in session |
| Auth field | — | `AUTH_TOKEN` (default) or `API_KEY`; **wrong one = 401** |
| API key | `ANTHROPIC_AUTH_TOKEN` or `ANTHROPIC_API_KEY` | Value for the chosen auth field |
| opus → model | `ANTHROPIC_DEFAULT_OPUS_MODEL` | Three-tier model mapping; haiku also covers background tasks — **must be set** |
| sonnet → model | `ANTHROPIC_DEFAULT_SONNET_MODEL` | |
| haiku → model | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | |
| effort level | `CLAUDE_CODE_EFFORT_LEVEL` | `low`–`max`; `auto` = model default; empty = unset. Third parties may not honor it |
| Disable nonessential traffic | `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` disables nonessential Claude Code traffic; empty = unset. Zhipu GLM defaults to `1` |
| Context window | `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | Fill with a Claude Code-supported window value; empty = unset |

> [!NOTE]
> CC-X **deliberately does not set** `ANTHROPIC_MODEL`. Use `/model opus|sonnet|haiku` in-session;
> the mapping table translates to the provider's real model name.

### Auth field: AUTH_TOKEN vs API_KEY

| Option | Request header | Used by |
|---|---|---|
| `AUTH_TOKEN` (default) | `Authorization: Bearer <key>` | Most third-party relays |
| `API_KEY` | `x-api-key: <key>` | The official API, and a few relays |

### Pre-seeded profiles

| Profile | BASE_URL | OPUS / SONNET | HAIKU (incl. background) | effort | Extra env |
|---|---|---|---|---|---|
| Official | empty (logged-in) | — | — | — | — |
| DeepSeek | `https://api.deepseek.com/anthropic` | `deepseek-v4-pro` | `deepseek-v4-flash` | `max` (recommended) | — |
| Zhipu GLM | `https://open.bigmodel.cn/api/anthropic` | `GLM-4.7` | `glm-4.5-air` | — | `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1` |
| Xiaomi MiMo | `https://api.xiaomimimo.com/anthropic` | `mimo-v2.5-pro` | `mimo-v2.5-pro` | — | — |

> [!NOTE]
> Model names change as providers update. Xiaomi MiMo has both pay-as-you-go and TokenPlan
> endpoints; you pick one when selecting the provider.

### Advanced

- **Multiple accounts**: create multiple profiles from the same provider — names auto-suffix
  with ` 2`, ` 3`… Use **Note** to tell them apart, shown as "Provider — Note".
- **Custom providers**: `presets.json` is the provider catalog; add a JSON entry to offer a new
  one, no code change. Drop `~/.cc-mini/presets.json` to override the shipped catalog.
- **First-launch login prompt**: before launching a third-party profile, CC-X reads
  `hasCompletedOnboarding` from `~/.claude.json` and prints a non-blocking hint when onboarding
  is unfinished. It does not write the file; dismiss or skip the Claude Code prompt if it appears.
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
- **No Claude Code config file is ever modified.** Before third-party launches, CC-X only reads
  the onboarding field in `~/.claude.json` to decide whether to print a hint.

CC-X only touches these 8 "managed" variables (and clears the ones a target profile doesn't use):
`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_API_KEY`, `ANTHROPIC_DEFAULT_OPUS_MODEL`,
`ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `CLAUDE_CODE_EFFORT_LEVEL`,
`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`.

> [!TIP]
> To change `settings.json`, use Claude Code's own `/update-config` and describe what you want
> in natural language (e.g. "allow npm commands") — safer than letting an external tool rewrite it.

---

## 🤔 Is CC-X right for you?

**Yes** — if you live in the terminal, often keep several terminals open, have been burned by a
config-wrecking switcher, and just want the one thing: **switch APIs**.

**No** — if you need any of these, [cc-switch](https://github.com/farion1231/cc-switch) fits better
(it's an excellent full-featured GUI; CC-X takes the opposite, minimal route):

- You need to manage MCP, hooks, plugins, or multiple CLIs
- You want a GUI, or automatic config migration / backup
- You only use the official API and never switch → you simply don't need CC-X

### CC-X vs cc-switch

| | CC-X (`xx`) | cc-switch |
|---|---|---|
| Form | Terminal command (lightweight) | Desktop GUI (full-featured) |
| Scope | Just API switching | API + MCP + multiple CLIs + prompts… |
| Touches config? | **Never** (env vars only) | Rewrites config from its own DB |
| Can lose MCP? | **Physically impossible** | Users have reported it |
| Parallel terminals | **Native** (process isolation) | Global switch; sessions can clash |

---

## 🧭 Design philosophy

> CC-X cares more about boundaries than features.

Claude Code already has its own config system, MCP ecosystem, and session state. CC-X is not trying to become a control panel above it, or to copy user config into another database. It stands at one narrow point before Claude Code starts: prepare the 8 managed environment variables, then let Claude Code run.

That constraint is deliberate: no writes to Claude Code config files, no MCP management, no automatic migration, no resident background controller. If process environment variables can solve it, CC-X avoids global files; if a choice matters, the user makes it explicitly. Doing less keeps the failure surface small.

Issues / PRs are welcome, but the direction is clear: **make switching steadier, clearer, and less intrusive** before adding broader management power. Anything that writes a Claude Code config file will not be accepted.

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

## 🗑️ Uninstall

<details>
<summary>Show uninstall steps</summary>

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

</details>

---

## 📄 License

[MIT](LICENSE)
