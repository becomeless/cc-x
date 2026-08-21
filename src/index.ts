#!/usr/bin/env node
/**
 * cc-x —— Claude Code API 切换器（命令：xx）
 *
 * 入口：解析 CLI 参数 → 分派到 CLI 路径（--list / xx <name> / -s）或交互菜单（TUI）。
 *
 * 铁律：API、密钥和模型映射只走 9 个受管环境变量（effort 走 --effort 启动参数，见 env/session.ts effortArgs）；唯一配置写入是用户主动触发的 onboarding 布尔字段。详见 AGENTS.md / plan §2。
 */
import { createRequire } from 'node:module';
import { Command, Option } from 'commander';

import { launchSession } from './actions.js';
import { getProviderState, loadStore, peekStoreLang, resolveStorePaths, StoreError, type Provider, type Store, type StorePaths } from './config/store.js';
import { loadPresets } from './config/presets.js';
import { setDefault } from './env/default.js';
import { providerDisplayName, resolveLang, setLang, T } from './i18n/index.js';
import { currentTerminalLine } from './runtime-info.js';
import { openMenu, profileRows } from './ui/menus.js';
import { runUpdate } from './update/update.js';

const require = createRequire(import.meta.url);
const pkg = require('../package.json') as { version: string };

type DefaultScope = 'user' | 'process';
type Lang = 'zh' | 'en';

interface GlobalOpts {
  session?: boolean;
  list?: boolean;
  storeDir?: string;
  defaultScope: DefaultScope;
  lang?: Lang;
}

/** 从 argv 里轻量预读一个带值参数（`--name v` 或 `--name=v`），用于 parse 前定语言。 */
function peekArg(argv: string[], name: string): string | undefined {
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === name) return argv[i + 1];
    if (a && a.startsWith(`${name}=`)) return a.slice(name.length + 1);
  }
  return undefined;
}

function main(): void {
  // help/version 在 commander parse 时就打印，故须**在建命令前**定好语言：
  // --lang > providers.json 的 lang（只读探测，不生成文件）> 环境 > 默认 zh。
  const argv = process.argv;
  const earlyLang = peekArg(argv, '--lang');
  const lang = earlyLang === 'en' ? 'en' : earlyLang === 'zh' ? 'zh' : undefined;
  const storeLang = peekStoreLang(resolveStorePaths(peekArg(argv, '--store-dir')));
  setLang(resolveLang(lang, storeLang));

  const program = new Command();
  program
    .name('xx')
    .description(T('cli.desc'))
    .helpOption('-h, --help', T('cli.opt.help'))
    .version(pkg.version, '-v, --version', T('cli.opt.version'))
    .argument('[name]', T('cli.arg.name'))
    .option('-s, --session', T('cli.opt.session'))
    .option('-l, --list', T('cli.opt.list'))
    .option('--store-dir <dir>', T('cli.opt.storeDir'))
    .addOption(
      // [P1] 严格校验：拼错（如 proces）直接报错退出，不静默回退到危险的持久化路径。
      new Option('--default-scope <scope>', T('cli.opt.defaultScope')).choices(['user', 'process']).default('user'),
    )
    .addOption(new Option('--lang <lang>', T('cli.opt.lang')).choices(['zh', 'en']))
    // 子命令优先于顶层 [name] 位置参数：`xx update` 命中这里而不是找名为 update 的配置。
    .command('update')
    .description(T('cli.cmd.update'))
    .action(async () => {
      await runUpdate(pkg.version);
    });

  program
    .action(async (name: string | undefined, raw: GlobalOpts) => {
      await dispatch(name, normalizeOpts(raw));
    });

  void program.parseAsync();
}

function normalizeOpts(raw: GlobalOpts): GlobalOpts {
  const scope = raw.defaultScope === 'process' ? 'process' : 'user';
  const lang = raw.lang === 'en' ? 'en' : raw.lang === 'zh' ? 'zh' : undefined;
  return { ...raw, defaultScope: scope, lang };
}

/** 按参数把请求分派到对应路径。 */
async function dispatch(name: string | undefined, opts: GlobalOpts): Promise<void> {
  const paths = resolveStorePaths(opts.storeDir);
  let store: Store;
  try {
    store = loadStore(paths);
  } catch (e) {
    if (e instanceof StoreError) {
      // store 不可用，按 --lang/环境定语言（拿不到 store.lang）。绝不重建、不动用户数据。
      setLang(resolveLang(opts.lang));
      const head = e.kind === 'read' ? 'error.storeRead' : e.kind === 'format' ? 'error.storeFormat' : 'error.storeCorrupt';
      console.error(`  ${T(head, e.file)}`);
      console.error(`  ${T('error.storeCorruptHint')}`);
      if (e.kind === 'parse' || e.kind === 'format') {
        console.error(`  ${T('error.storeBackupHint', backupCommand(e.file))}`);
      }
      process.exitCode = 1;
      return;
    }
    throw e;
  }
  // 语言：--lang > providers.json lang > 环境 > 默认 zh。在产出任何文案前先定好。
  setLang(resolveLang(opts.lang, store.lang));

  if (opts.list) {
    runList(store);
    return;
  }
  if (name) {
    const target = findProviderForCli(store, name);
    if (!target) {
      console.error(`  ${T('error.notFound', name)}`);
      console.error(`  ${T('error.existing', store.providers.map((p) => providerDisplayName(p)).join(', '))}`);
      console.error(`  ${T('error.notFoundHint')}`);
      process.exitCode = 1;
      return;
    }
    if (opts.session) launchSession(target);
    else runDefault(paths, store, target, opts.defaultScope);
    return;
  }
  await openMenu(paths, store, opts.defaultScope, pkg.version, loadPresets(opts.storeDir));
}

function findProviderForCli(store: Store, name: string): Provider | undefined {
  return store.providers.find((p) => p.name === name) ?? store.providers.find((p) => providerDisplayName(p) === name);
}

/** `--list`：列出所有配置及状态。官方档显示名走 i18n（评审①），其余原样。 */
function runList(store: Store): void {
  const cur = store.providers.find((p) => p.name === store.current);
  console.log('');
  console.log(`  ${T('list.default', cur ? providerDisplayName(cur) : store.current)}`);
  console.log(`  ${currentTerminalLine(store)}`);
  for (const line of profileRows(store.providers, store.current)) {
    console.log(`  ${line}`);
  }
  console.log('');
}

/** 设为默认：写用户环境变量（或 dry-run）+ 更新 store.current。 */
function runDefault(paths: StorePaths, store: Store, p: Provider, scope: DefaultScope): void {
  warnIfNoKeyForDefault(p);
  const name = providerDisplayName(p);
  const r = setDefault(paths, store, p, scope);

  if (r.dryRun) {
    console.log(`  ${T('default.done', name)}`);
    console.log(`  ${T('default.dryRun')}`);
    console.log(`  ${T('default.hintSession', quoteArg(p.name))}`);
    return;
  }
  if (r.windows && !r.windows.ok) {
    console.error(`  ${T('default.failed', r.windows.error ?? '')}`);
    process.exitCode = 1;
    return;
  }
  if (r.unix?.unsupported) {
    console.error(`  ${T('default.fishUnsupported')}`);
    return;
  }
  console.log(`  ${T('default.done', name)}`);
  if (r.unix) console.log(`  ${T('default.unixWrote', r.unix.file)}`);
  console.log(`  ${T('default.hintSession', quoteArg(p.name))}`);
}

function warnIfNoKeyForDefault(p: Provider): void {
  if (getProviderState(p).key === 'noKey') {
    console.log(`  ${T('default.noKey', providerDisplayName(p))}`);
  }
}

function backupCommand(file: string): string {
  const dst = `${file}.bak`;
  if (process.platform === 'win32') {
    return `Copy-Item -LiteralPath ${psQuote(file)} -Destination ${psQuote(dst)}`;
  }
  return `cp ${shQuote(file)} ${shQuote(dst)}`;
}

function psQuote(s: string): string {
  return `'${s.replaceAll("'", "''")}'`;
}

function shQuote(s: string): string {
  return `'${s.replaceAll("'", `'"'"'`)}'`;
}

function quoteArg(s: string): string {
  return `"${s.replaceAll('"', '\\"')}"`;
}

main();
