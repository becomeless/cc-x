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
  <a href="#install">Install</a> · <a href="#60-second-quick-start">Quick Start</a> · <a href="#two-modes-the-key-concept">Concepts</a> · <a href="#configuration">Config</a> · <a href="#faq">FAQ</a>
</p>

---

> `xx` — one command to switch Claude Code between APIs. **Zero config risk.**

Switching Claude Code between the official account and third-party APIs means juggling
environment variables — or trusting a tool that rewrites your Claude config. CC-X takes a
different path: **switching happens purely at the environment-variable layer.** It never
reads or writes any Claude Code config file. Your MCP, plugins, hooks — it won't touch them.

```text
  CC-X v0.4.12     Default: Official

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

## Install

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

> With the menu's "Update check" set to "notify", CC-X shows a banner at the top of the menu
> when a new version is out, with the matching upgrade command for your platform.

---

## 60-second quick start

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

## Two modes (the key concept)

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

## When CC-X is NOT the right tool

- You need to manage MCP, hooks, plugins, or multiple CLIs → use [cc-switch](https://github.com/farion1231/cc-switch)
- You only use the official API, never switch → you don't need CC-X
- You want automatic config migration/backup → that's outside CC-X's scope

CC-X cares more about boundaries than features. It does one thing: **switch APIs**.

---

## CC-X vs cc-switch

cc-switch is an excellent full-featured GUI; CC-X takes the opposite, minimal approach.

| | CC-X (`xx`) | cc-switch |
|---|---|---|
| Form | Terminal command (lightweight) | Desktop GUI (full-featured) |
| Scope | Just API switching | API + MCP + multiple CLIs + prompts… |
| Touches config? | **Never** (env vars only) | Rewrites config from its own DB |
| Can lose MCP? | **Physically impossible** | Users have reported it |
| Parallel terminals | **Native** (process isolation) | Global switch; sessions can clash |

- → **CC-X**: terminal natives, parallel-session runners, anyone burned by a config-wrecking switcher, "just switch the API" people
- → **cc-switch**: GUI preference, all-in-one MCP + multi-CLI management

---

## Design philosophy

> CC-X cares more about boundaries than features.

Claude Code already has its own config system, MCP ecosystem, and session state. CC-X is not trying to become a control panel above it, or to copy user config into another database. It stands at one narrow point before Claude Code starts: prepare the 10 managed environment variables, then let Claude Code run.

That constraint is deliberate: no writes to Claude Code config files, no MCP management, no automatic migration, no resident background controller. If process environment variables can solve it, CC-X avoids global files; if a choice matters, the user makes it explicitly. Doing less keeps the failure surface small.

Issues / PRs are welcome, but the direction is clear: **make switching steadier, clearer, and less intrusive** before adding broader management power. Anything that writes a Claude Code config file will not be accepted.

---

## Configuration

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
| subagent → model | `CLAUDE_CODE_SUBAGENT_MODEL` | Model for subagents/agent teams; **empty = official default (inherit main model)**. To save cost, fill `haiku` (alias, follows this profile's haiku tier) or a concrete model ID |
| ↻ Fetch models | — | Pulls available models from the provider API; pick one, optionally attach `[1m]`, then apply to a tier. Falls back to manual input on failure |
| effort level | `CLAUDE_CODE_EFFORT_LEVEL` | `low`–`max`; `auto` = model default; empty = unset. Third parties may not honor it |
| Disable nonessential traffic | `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` disables nonessential Claude Code traffic; empty = unset. Zhipu GLM defaults to `1` |

> CC-X **deliberately does not set** `ANTHROPIC_MODEL`. Use `/model opus|sonnet|haiku|fable` in-session;
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

> Model names change as providers update. Xiaomi MiMo has both pay-as-you-go and TokenPlan
> endpoints; you pick one when selecting the provider.
> In "Fetch models", models known to support 1M context (e.g. `deepseek-v4-pro`, `glm-5.2`,
> `mimo-v2.5-pro`) are marked `[1M]`; selecting one can auto-append the `[1m]` suffix — no typing.

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

## Data & files

- **Profiles (plaintext keys, keep local)**: `~/.cc-mini/providers.json` (also holds `lang` and `update`)
- **Provider catalog**: shipped `presets.json`; override at `~/.cc-mini/presets.json`
- **"Set default" writes user environment variables** (not Claude config files):
  - Windows → registry `HKCU\Environment` + one change broadcast
  - Unix → `# >>> xx >>>` … `# <<< xx <<<` marker block in shell startup file (idempotent rewrite, chosen by `$SHELL`)
  - Same semantics either way: **only affects new terminals**; switching to "Official" clears all managed vars
- **No Claude Code config file is ever modified.** Before third-party launches, CC-X only reads
  the onboarding field in `~/.claude.json` to decide whether to print a hint.

CC-X only touches these 10 "managed" variables (and clears the ones a target profile doesn't use):
`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_API_KEY`, `ANTHROPIC_DEFAULT_OPUS_MODEL`,
`ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `ANTHROPIC_DEFAULT_FABLE_MODEL`,
`CLAUDE_CODE_SUBAGENT_MODEL`, `CLAUDE_CODE_EFFORT_LEVEL`, `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`.

> 💡 To change `settings.json`, use Claude Code's own `/update-config` and describe what you want
> in natural language (e.g. "allow npm commands") — safer than letting an external tool rewrite it.

---

## FAQ

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

## Uninstall

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

## License

[MIT](LICENSE)
