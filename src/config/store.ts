/**
 * 配置存取层：读写 ~/.cc-mini/providers.json，生成默认配置，构造 env map。
 *
 * 铁律：这里写的是工具自己的数据文件（providers.json），**绝不**碰 ~/.claude/*。
 * 格式与现版 PowerShell 完全兼容；写入 UTF-8 无 BOM、2 空格缩进。
 */
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

import { KNOWN_KEYS, type Lang, type Provider, type Store } from './types.js';

export { KNOWN_KEYS } from './types.js';
export type { Lang, ManagedKey, Provider, Store } from './types.js';

export interface StorePaths {
  dir: string;
  file: string;
}

/** 解析存储路径。`storeDir` 来自 --store-dir（测试用），默认 ~/.cc-mini。 */
export function resolveStorePaths(storeDir?: string): StorePaths {
  const dir = storeDir && storeDir.trim() ? storeDir : join(homedir(), '.cc-mini');
  return { dir, file: join(dir, 'providers.json') };
}

const nonEmpty = (v: unknown): v is string => typeof v === 'string' && v.trim() !== '';

/**
 * 默认配置：官方 + DeepSeek + 智谱GLM + 小米MiMo（密钥空）。
 * 官方档带 `builtin: 'official'`（评审①），其它供应商是专有名词、不翻译。
 */
export function defaultStore(): Store {
  return {
    current: '官方',
    lang: 'zh',
    providers: [
      { name: '官方', note: '', builtin: 'official', env: {} },
      {
        name: 'DeepSeek',
        note: '',
        env: {
          ANTHROPIC_BASE_URL: 'https://api.deepseek.com/anthropic',
          ANTHROPIC_DEFAULT_OPUS_MODEL: 'deepseek-v4-pro',
          ANTHROPIC_DEFAULT_SONNET_MODEL: 'deepseek-v4-pro',
          ANTHROPIC_DEFAULT_HAIKU_MODEL: 'deepseek-v4-flash',
          CLAUDE_CODE_EFFORT_LEVEL: 'max',
        },
      },
      {
        name: '智谱GLM',
        note: '',
        env: {
          ANTHROPIC_BASE_URL: 'https://open.bigmodel.cn/api/anthropic',
          ANTHROPIC_DEFAULT_OPUS_MODEL: 'GLM-4.7',
          ANTHROPIC_DEFAULT_SONNET_MODEL: 'GLM-4.7',
          ANTHROPIC_DEFAULT_HAIKU_MODEL: 'glm-4.5-air',
          CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1',
        },
      },
      {
        name: '小米MiMo',
        note: '',
        env: {
          ANTHROPIC_BASE_URL: 'https://api.xiaomimimo.com/anthropic',
          ANTHROPIC_DEFAULT_OPUS_MODEL: 'mimo-v2.5-pro',
          ANTHROPIC_DEFAULT_SONNET_MODEL: 'mimo-v2.5-pro',
          ANTHROPIC_DEFAULT_HAIKU_MODEL: 'mimo-v2.5-pro',
        },
      },
    ],
  };
}

/**
 * 把任意解析结果规整为合法 Store。
 *
 * 字段级**宽松容错**（缺 lang / 缺 note / 缺 builtin / 缺 env 都不报错），保持既有数据格式兼容；
 * 但结构级**严格校验**：顶层必须是对象、`providers` 必须是数组、每个配置的 name/env 结构必须合法
 * —— 否则抛 StoreError('format')。
 * 这是为了堵住「语法合法但结构损坏的 JSON 被静默规整成空 providers、用户一保存就覆盖丢数据」的坑（[P1]）。
 */
function normalizeStore(raw: unknown, file: string): Store {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new StoreError('format', file);
  const obj = raw as Record<string, unknown>;
  if (!Array.isArray(obj.providers)) throw new StoreError('format', file);
  const providers: Provider[] = obj.providers.map((p) => normalizeProvider(p, file));
  const lang: Lang | undefined = obj.lang === 'en' ? 'en' : obj.lang === 'zh' ? 'zh' : undefined;
  const current = typeof obj.current === 'string' ? obj.current : (providers[0]?.name ?? '');
  // 只认已知模式，未知值视为关闭（不回写）。字段顺序须与 Go 版一致：…providers, update?
  const update = obj.update === 'notify' || obj.update === 'auto' ? obj.update : undefined;
  return { current, ...(lang ? { lang } : {}), providers, ...(update ? { update } : {}) };
}

function normalizeProvider(raw: unknown, file: string): Provider {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new StoreError('format', file);
  const p = raw as Record<string, unknown>;
  if (typeof p.name !== 'string') throw new StoreError('format', file);
  if (p.note !== undefined && typeof p.note !== 'string') throw new StoreError('format', file);
  if (p.builtin !== undefined && typeof p.builtin !== 'string') throw new StoreError('format', file);
  if (p.env !== undefined && (!p.env || typeof p.env !== 'object' || Array.isArray(p.env))) {
    throw new StoreError('format', file);
  }
  const name = p.name;
  const note = typeof p.note === 'string' ? p.note : '';
  const builtin = typeof p.builtin === 'string' ? p.builtin : undefined;
  const env: Record<string, string> = {};
  if (p.env) {
    for (const [k, v] of Object.entries(p.env as Record<string, unknown>)) {
      if (typeof v !== 'string') throw new StoreError('format', file);
      env[k] = v;
    }
  }
  return { name, note, ...(builtin ? { builtin } : {}), env };
}

/**
 * 配置文件存在但不可用时抛出。调用方据此给出友好提示并退出，
 * **绝不**静默重建/覆盖——那会清掉用户的明文密钥（违背「不碰用户数据」初心）。
 *   - `read`  ：读文件失败（权限/磁盘/同名目录 EISDIR …）[P2]
 *   - `parse` ：JSON 语法损坏 [上一轮已修]
 *   - `format`：JSON 语法合法但结构损坏（顶层非对象 / providers 非数组）[P1]
 */
export type StoreErrorKind = 'read' | 'parse' | 'format';

export class StoreError extends Error {
  constructor(
    public readonly kind: StoreErrorKind,
    public readonly file: string,
  ) {
    super(`store ${kind} error: ${file}`);
    this.name = 'StoreError';
  }
}

/** 读配置；文件不存在则生成默认并落盘后返回；文件不可用则抛 StoreError（绝不覆盖）。 */
export function loadStore(paths: StorePaths): Store {
  let text: string;
  try {
    text = readFileSync(paths.file, 'utf-8'); // [P2] 权限/磁盘/EISDIR 等
  } catch (e) {
    if ((e as NodeJS.ErrnoException).code !== 'ENOENT') throw new StoreError('read', paths.file);
    const store = defaultStore();
    saveStore(paths, store);
    return store;
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new StoreError('parse', paths.file);
  }
  return normalizeStore(parsed, paths.file); // [P1] 结构校验在内部，失败抛 'format'
}

/**
 * 只读探测 lang，**不生成文件**（用于 --help/--version 在 parse 前定语言，避免副作用）。
 * 文件不存在/解析失败都返回 undefined。
 */
export function peekStoreLang(paths: StorePaths): Lang | undefined {
  try {
    if (!existsSync(paths.file)) return undefined;
    const raw = JSON.parse(readFileSync(paths.file, 'utf-8')) as { lang?: unknown };
    return raw.lang === 'en' ? 'en' : raw.lang === 'zh' ? 'zh' : undefined;
  } catch {
    return undefined;
  }
}

/** 写配置：UTF-8 无 BOM、2 空格缩进（Node 默认 utf-8 不带 BOM，与现版一致）。 */
export function saveStore(paths: StorePaths, store: Store): void {
  if (!existsSync(paths.dir)) mkdirSync(paths.dir, { recursive: true });
  writeFileSync(paths.file, `${JSON.stringify(store, null, 2)}\n`, 'utf-8');
}

/**
 * 是否官方档。优先认稳定标识 `builtin === 'official'`（评审①）；
 * 老文件没有 builtin 时，仅将「中文名为官方 + env 为空」视为官方档。
 * 这样旧数据仍兼容，而手动填了第三方地址/密钥的同名配置不会被误判为登录态。
 */
export function isOfficial(p: Provider): boolean {
  if (p.builtin) return p.builtin === 'official';
  return p.name === '官方' && Object.keys(p.env).length === 0;
}

/**
 * [P1] 编辑保存后修正身份：官方档（builtin='official' = 登录态、空 env）一旦被配成真实第三方
 * （env 非空，有了 base/key 等），就清掉 builtin —— 否则会继续被当登录态、跳过缺密钥警告。
 */
export function reconcileBuiltin(p: Provider): void {
  if (p.builtin === 'official' && Object.keys(p.env).length > 0) {
    delete p.builtin;
  }
}

/** 删除配置等操作后修正默认指向：优先剩余官方档，其次第一项；没有配置则置空。 */
export function reconcileCurrent(store: Store): void {
  if (store.providers.some((p) => p.name === store.current)) return;
  store.current = (store.providers.find(isOfficial) ?? store.providers[0])?.name ?? '';
}

/** 取配置的 env map（即 provider.env，保证非空对象）。 */
export function getProviderEnvMap(p: Provider): Record<string, string> {
  return p.env ?? {};
}

/**
 * 由一组字段构造 provider.env：按 KNOWN_KEYS 顺序、丢弃空白值。
 * 对齐现版 Build-ProviderEnv。`[1m]` 等后缀属于自由文本，原样保留（见 plan §3.1.1）。
 */
export function buildProviderEnv(fields: Record<string, string | undefined>): Record<string, string> {
  const env: Record<string, string> = {};
  for (const key of KNOWN_KEYS) {
    const v = fields[key];
    if (nonEmpty(v)) env[key] = v.trim();
  }
  return env;
}

/** 配置的密钥状态（语义枚举，不含界面文案；翻译交给 i18n 层，评审①）。 */
export type KeyState = 'official' | 'noKey' | 'apiKey' | 'hasToken';

export interface ProviderState {
  key: KeyState;
  effort?: string;
}

export function getProviderState(p: Provider): ProviderState {
  const map = getProviderEnvMap(p);
  const effort = nonEmpty(map.CLAUDE_CODE_EFFORT_LEVEL) ? map.CLAUDE_CODE_EFFORT_LEVEL : undefined;
  if (isOfficial(p)) return { key: 'official', ...(effort ? { effort } : {}) };
  const hasTok = nonEmpty(map.ANTHROPIC_AUTH_TOKEN);
  const hasKey = nonEmpty(map.ANTHROPIC_API_KEY);
  const key: KeyState = !hasTok && !hasKey ? 'noKey' : hasKey ? 'apiKey' : 'hasToken';
  return { key, ...(effort ? { effort } : {}) };
}

/** 按 name 找配置。 */
export function findProvider(store: Store, name: string): Provider | undefined {
  return store.providers.find((p) => p.name === name);
}

/**
 * 名称去重：同名被【其它】配置占用时追加 “ 2/3/…”。`exclude` 是正在编辑的本条（排除自身）。
 * 对齐现版 Resolve-UniqueName。
 */
export function resolveUniqueName(store: Store, name: string, exclude?: Provider): string {
  const existing = store.providers.filter((p) => p !== exclude).map((p) => p.name);
  if (!existing.includes(name)) return name;
  let i = 2;
  while (existing.includes(`${name} ${i}`)) i += 1;
  return `${name} ${i}`;
}

/** 语言：providers.json 的 lang 字段，缺省视为 zh。 */
export function getLang(store: Store): Lang {
  return store.lang === 'en' ? 'en' : 'zh';
}

export function setLang(store: Store, lang: Lang): void {
  store.lang = lang;
}
