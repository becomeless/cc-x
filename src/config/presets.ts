/**
 * 供应商目录（presets）加载层。
 *
 * 加载优先级（评审⑤）：
 *   1) 用户目录 ~/.cc-mini/presets.json（可选，用户加供应商不必动 node_modules）
 *   2) 包内随发布的 presets.json（dist 的上级目录）
 *   3) 内置常量 BUILTIN_PRESETS（最后兜底，等价于现版 $BuiltinPresetsJson）
 *
 * 任一步解析失败/格式不对就跌落到下一步，绝不抛错中断启动。
 */
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { resolveStorePaths } from './store.js';
import type { Preset, PresetEnv, PresetModels, PresetUrl } from './types.js';

export type { Preset } from './types.js';

/** 内置兜底目录（镜像仓库 presets.json）。第三方供应商**绝不**预置 `[1m]`（见 plan §3.1.1）。 */
export const BUILTIN_PRESETS: Preset[] = [
  {
    name: 'DeepSeek',
    auth: 'AUTH_TOKEN',
    effort: 'max',
    urls: [{ label: 'Anthropic 兼容', url: 'https://api.deepseek.com/anthropic' }],
    models: { opus: 'deepseek-v4-pro', sonnet: 'deepseek-v4-pro', haiku: 'deepseek-v4-flash' },
    modelsApi: 'https://api.deepseek.com/models', // Anthropic 兼容端点未实现 /v1/models，模型列表在 OpenAI 风格端点
    models1m: ['deepseek-v4-pro'],
  },
  {
    name: '智谱GLM',
    auth: 'AUTH_TOKEN',
    urls: [{ label: 'Anthropic 兼容', url: 'https://open.bigmodel.cn/api/anthropic' }],
    models: { opus: 'GLM-4.7', sonnet: 'GLM-4.7', haiku: 'glm-4.5-air' },
    env: {
      CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1',
    },
    modelsApi: 'https://open.bigmodel.cn/api/anthropic/v1/models',
    models1m: ['glm-5.2'],
  },
  {
    name: '小米MiMo',
    auth: 'AUTH_TOKEN',
    urls: [
      { label: '按量付费API', url: 'https://api.xiaomimimo.com/anthropic' },
      { label: 'TokenPlan', url: 'https://token-plan-cn.xiaomimimo.com/anthropic' },
    ],
    models: { opus: 'mimo-v2.5-pro', sonnet: 'mimo-v2.5-pro', haiku: 'mimo-v2.5-pro' },
    // MiMo 的 models 端点是 OpenAI 风格 /v1/models（不带 /anthropic 前缀），按 base 前缀匹配各自端点
    modelsApiMap: {
      'https://api.xiaomimimo.com/anthropic': 'https://api.xiaomimimo.com/v1/models',
      'https://token-plan-cn.xiaomimimo.com/anthropic': 'https://token-plan-cn.xiaomimimo.com/v1/models',
    },
    models1m: ['mimo-v2.5-pro'],
  },
  {
    name: 'Kimi (月之暗面)',
    auth: 'AUTH_TOKEN',
    effort: 'max',
    urls: [{ label: 'Anthropic 兼容', url: 'https://api.moonshot.cn/anthropic' }],
    models: { opus: 'kimi-k3', sonnet: 'kimi-k3', haiku: 'kimi-k3', fable: 'kimi-k3' },
    models1m: ['kimi-k3'],
  },
  {
    name: 'MiniMax',
    auth: 'AUTH_TOKEN',
    urls: [
      { label: '国内站', url: 'https://api.minimaxi.com/anthropic' },
      { label: '国际站', url: 'https://api.minimax.io/anthropic' },
    ],
    models: { opus: 'MiniMax-M3', sonnet: 'MiniMax-M3', haiku: 'MiniMax-M3', fable: 'MiniMax-M3' },
    models1m: ['MiniMax-M3'],
  },
  {
    name: 'OpenRouter',
    auth: 'API_KEY',
    urls: [{ label: 'Anthropic 兼容', url: 'https://openrouter.ai/api' }],
    models: { opus: 'anthropic/claude-opus-5', sonnet: 'anthropic/claude-sonnet-5', haiku: 'anthropic/claude-haiku-4-5', fable: 'anthropic/claude-fable-5' },
  },
  {
    name: '百度千帆',
    auth: 'AUTH_TOKEN',
    urls: [{ label: 'Coding Plan', url: 'https://qianfan.baidubce.com/anthropic/coding' }],
    models: { opus: 'qianfan-code-latest', sonnet: 'qianfan-code-latest', haiku: 'qianfan-code-latest', fable: 'qianfan-code-latest' },
    env: {
      CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1',
    },
  },
  {
    name: '阿里百炼',
    auth: 'AUTH_TOKEN',
    urls: [
      { label: 'Token Plan（个人/团队）', url: 'https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic' },
      { label: '按量付费', url: 'https://dashscope.aliyuncs.com/apps/anthropic' },
    ],
    models: { opus: 'qwen3.8-max', sonnet: 'qwen3.8-max', haiku: 'qwen3.6-flash', fable: 'qwen3.8-max' },
    models1m: ['qwen3.8-max', 'qwen3.7-max', 'qwen3.7-plus'],
  },
  {
    name: '火山方舟',
    auth: 'AUTH_TOKEN',
    urls: [{ label: 'Coding Plan', url: 'https://ark.cn-beijing.volces.com/api/coding' }],
    models: { opus: 'doubao-seed-2.0-code', sonnet: 'doubao-seed-2.0-code', haiku: 'doubao-seed-2.0-code', fable: 'doubao-seed-2.0-code' },
  },
  {
    name: '官方Anthropic',
    auth: 'API_KEY',
    urls: [{ label: '(留空，用登录态)', url: '' }],
    models: {},
  },
];

const asString = (v: unknown): string => (typeof v === 'string' ? v : '');

function normalizeUrl(raw: unknown): PresetUrl {
  const u = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  return { label: asString(u.label), url: asString(u.url) };
}

function normalizeModels(raw: unknown): PresetModels {
  const m = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const models: PresetModels = {};
  if (typeof m.opus === 'string') models.opus = m.opus;
  if (typeof m.sonnet === 'string') models.sonnet = m.sonnet;
  if (typeof m.haiku === 'string') models.haiku = m.haiku;
  if (typeof m.fable === 'string') models.fable = m.fable;
  return models;
}

const PRESET_ENV_KEYS = ['CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC', 'CLAUDE_CODE_SUBAGENT_MODEL'] as const;

function normalizeEnv(raw: unknown): PresetEnv | undefined {
  const m = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const env: PresetEnv = {};
  for (const key of PRESET_ENV_KEYS) {
    if (typeof m[key] === 'string') env[key] = m[key];
  }
  return Object.keys(env).length > 0 ? env : undefined;
}

function normalizePreset(raw: unknown): Preset | undefined {
  const p = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const name = asString(p.name).trim();
  if (!name) return undefined; // 无名条目直接丢弃
  const auth: Preset['auth'] = p.auth === 'API_KEY' ? 'API_KEY' : 'AUTH_TOKEN';
  const urls = Array.isArray(p.urls) ? p.urls.map(normalizeUrl) : [];
  const preset: Preset = { name, auth, urls, models: normalizeModels(p.models) };
  if (typeof p.effort === 'string' && p.effort.trim()) preset.effort = p.effort.trim();
  const env = normalizeEnv(p.env);
  if (env) preset.env = env;
  if (typeof p.models_api === 'string' && p.models_api.trim()) preset.modelsApi = p.models_api.trim();
  if (p.models_api_map && typeof p.models_api_map === 'object') {
    const map: Record<string, string> = {};
    for (const [k, v] of Object.entries(p.models_api_map as Record<string, unknown>)) {
      if (typeof v === 'string' && v.trim()) map[k.trim()] = v.trim();
    }
    if (Object.keys(map).length > 0) preset.modelsApiMap = map;
  }
  if (Array.isArray(p.models_1m)) {
    const list = p.models_1m.map(asString).map((s) => s.trim()).filter(Boolean);
    if (list.length > 0) preset.models1m = list;
  }
  return preset;
}

/** 把任意解析结果规整为 Preset[]；非数组或全空则返回 undefined（让调用方跌落兜底）。 */
function normalizePresets(raw: unknown): Preset[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  const list = raw.map(normalizePreset).filter((p): p is Preset => p !== undefined);
  return list.length > 0 ? list : undefined;
}

/** 尝试读并解析一个 presets.json；任何问题都安静返回 undefined。 */
function tryLoadFile(file: string): Preset[] | undefined {
  try {
    if (!existsSync(file)) return undefined;
    return normalizePresets(JSON.parse(readFileSync(file, 'utf-8')));
  } catch {
    return undefined;
  }
}

/** 定位包内随发布的 presets.json（dist 的上级 = 包根；dev 下 = 仓库根）。 */
function packagePresetsFile(): string {
  // 本模块编译后位于 dist/config/presets.js → ../../presets.json = 包根。
  return fileURLToPath(new URL('../../presets.json', import.meta.url));
}

/**
 * 加载供应商目录。`storeDir` 来自 --store-dir（测试用），决定用户覆盖文件的位置。
 * 优先级见文件头注释。
 */
export function loadPresets(storeDir?: string): Preset[] {
  const userFile = join(resolveStorePaths(storeDir).dir, 'presets.json');
  return tryLoadFile(userFile) ?? tryLoadFile(packagePresetsFile()) ?? BUILTIN_PRESETS;
}
