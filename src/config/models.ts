/**
 * 拉取供应商模型列表（编辑表单「获取模型列表」用，对齐 Go 版 internal/presets/models.go）。
 *
 * 端点缺省推导 `{base}/v1/models`（Anthropic 风格），供应商可用 presets.json 的 `models_api` 覆盖
 * （MiMo 的 models 端点是 OpenAI 风格 /v1/models，不带 /anthropic 前缀）。
 * 响应兼容 Anthropic 风格（data[].id + display_name）与 OpenAI 风格（data[].id），
 * 并防御「HTTP 200 但 body 是业务错误」的情况（GLM 无 key 时返回 {code,msg}）。
 */
import type { Preset } from './types.js';

/** 模型列表里的一项。 */
export interface ModelInfo {
  id: string;
  displayName?: string;
}

const MODELS_TIMEOUT_MS = 10_000;

/** 拉取模型列表。baseUrl 是 Anthropic 兼容端点，apiKey 是认证令牌。endpoint 为空时推导。 */
export async function fetchModels(baseUrl: string, apiKey: string, endpoint?: string): Promise<ModelInfo[]> {
  const url = endpoint?.trim() || `${baseUrl.replace(/\/+$/, '')}/v1/models`;
  const res = await fetch(url, {
    method: 'GET',
    headers: {
      // 双认证头都带上：Anthropic 官方认 x-api-key，国产中转一般认 Authorization: Bearer。
      Authorization: `Bearer ${apiKey}`,
      'x-api-key': apiKey,
      'User-Agent': 'ccx',
    },
    signal: AbortSignal.timeout(MODELS_TIMEOUT_MS),
  });
  const body = await res.text();
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${extractErrMsg(body)}（${url}）`);
  }
  return parseModels(body);
}

/** 解析 models 响应，兼容 Anthropic / OpenAI 两种 data 格式。 */
export function parseModels(body: string): ModelInfo[] {
  let raw: { data?: Array<{ id?: unknown; display_name?: unknown }> };
  try {
    raw = JSON.parse(body);
  } catch {
    throw new Error(`响应不是 JSON: ${extractErrMsg(body)}`);
  }
  if (!Array.isArray(raw?.data)) {
    // 200 但无 data 数组：业务错误体（如 GLM 的 {code,msg}）。
    throw new Error(extractErrMsg(body));
  }
  const out: ModelInfo[] = [];
  const seen = new Set<string>();
  for (const d of raw.data ?? []) {
    if (typeof d !== 'object' || d === null) continue;
    const id = String(d.id ?? '').trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    const name = typeof d.display_name === 'string' ? d.display_name.trim() : '';
    out.push(name ? { id, displayName: name } : { id });
  }
  if (out.length === 0) {
    throw new Error('模型列表为空');
  }
  return out;
}

/** 判断模型是否命中供应商的 1M 支持表（前缀匹配，先剥掉已有的 [1m] 后缀，大小写不敏感）。 */
export function supports1M(p: Preset | undefined, modelId: string): boolean {
  if (!p?.models1m?.length) return false;
  const id = modelId.trim().replace(/\[1m\]$/, '').toLowerCase();
  return p.models1m.some((item) => id.startsWith(item.toLowerCase()));
}

/** 尽力从错误响应体提取可读信息（error.message / msg / message / detail）。 */
function extractErrMsg(body: string): string {
  const trimmed = body.trim();
  if (!trimmed) return '空响应';
  let obj: unknown;
  try {
    obj = JSON.parse(trimmed);
  } catch {
    return trimmed.length > 120 ? trimmed.slice(0, 120) : trimmed;
  }
  if (typeof obj !== 'object' || obj === null) return trimmed;
  const m = obj as Record<string, unknown>;
  for (const key of ['msg', 'message', 'error', 'detail']) {
    const v = m[key];
    if (typeof v === 'string' && v.trim()) return v.trim();
    if (v && typeof v === 'object') {
      const msg = (v as Record<string, unknown>).message;
      if (typeof msg === 'string' && msg.trim()) return msg.trim();
    }
  }
  return trimmed;
}
