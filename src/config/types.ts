/**
 * 数据层共享类型与常量。
 *
 * 数据格式必须与现版 PowerShell 完全兼容（老用户的 ~/.cc-mini/providers.json 能直接读）。
 */

/** 受管的 10 个环境变量（工具只动这些，其它一律不碰）。详见 plan §2。 */
export const KNOWN_KEYS = [
  'ANTHROPIC_BASE_URL',
  'ANTHROPIC_AUTH_TOKEN',
  'ANTHROPIC_API_KEY',
  'ANTHROPIC_DEFAULT_OPUS_MODEL',
  'ANTHROPIC_DEFAULT_SONNET_MODEL',
  'ANTHROPIC_DEFAULT_HAIKU_MODEL',
  'ANTHROPIC_DEFAULT_FABLE_MODEL',
  'CLAUDE_CODE_SUBAGENT_MODEL',
  'CLAUDE_CODE_EFFORT_LEVEL',
  'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC',
] as const;

export type ManagedKey = (typeof KNOWN_KEYS)[number];

export type Lang = 'zh' | 'en';

/**
 * 一个「配置」（profile）。`name` 是唯一键（current / xx <name> / 删除都靠它）。
 *
 * `builtin`：稳定的内部标识，**与显示名解耦**（架构师评审①）。官方档 = `'official'`。
 * 这样界面切英文时，显示名可翻译成 "Official"，而代码判断仍认 `builtin`、数据主键 `name` 不变。
 * 老文件没有该字段：仅用 `name === '官方' && env 为空` 兜底判定（见 store.ts `isOfficial`）。
 */
export interface Provider {
  name: string;
  note?: string;
  builtin?: string;
  env: Record<string, string>;
}

/**
 * ~/.cc-mini/providers.json 的顶层结构。`lang` 为新增字段，旧文件缺省视为 zh。
 * `update`：更新检查模式，'notify'=提醒；缺省=关闭（默认，不写）。字段顺序须与 Go 版一致（…providers, update?）。
 */
export interface Store {
  current: string;
  lang?: Lang;
  providers: Provider[];
  update?: string;
}

/** presets.json 里一个供应商的某个 API 地址（可多个，多个时让用户选）。 */
export interface PresetUrl {
  label: string;
  url: string;
}

/** 四档模型映射（可部分为空）。 */
export interface PresetModels {
  opus?: string;
  sonnet?: string;
  haiku?: string;
  fable?: string;
}

/** 供应商目录可预填的额外受管变量。不接受鉴权字段，避免把 presets 变成密钥载体。 */
export type PresetEnv = Partial<Record<'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC' | 'CLAUDE_CODE_SUBAGENT_MODEL', string>>;

/** presets.json 里的一个「供应商」（provider）目录条目。 */
export interface Preset {
  name: string;
  auth: 'AUTH_TOKEN' | 'API_KEY';
  urls: PresetUrl[];
  models: PresetModels;
  effort?: string;
  env?: PresetEnv;
  /** 模型列表端点；缺省推导 `{base}/v1/models`（MiMo 路径不同需显式）。 */
  modelsApi?: string;
  /** 支持 1M 上下文的模型 ID 前缀表（`[1m]` 后缀默认开关）。 */
  models1m?: string[];
}
