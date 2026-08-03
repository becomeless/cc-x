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

/** 认证方式（与编辑表单的 auth 选择一致；决定模型列表请求携带哪个认证头）。 */
export type AuthMode = 'AUTH_TOKEN' | 'API_KEY';

/** 模型档位：四个固定模型映射档位行。 */
export type ModelSlot = 'opus' | 'sonnet' | 'haiku' | 'fable';

/** 档位是否允许自动附加 [1m] 后缀：Claude Code 支持通过 [1m] 标记扩展上下文，并在当前运行时中
 *  识别 opus/sonnet/fable；第三方映射到 fable 档的模型需要该标记声明 1M 能力（无标记按 200K 管理）。
 *  haiku 没有对应的文档和运行时依据，不自动附加。是否真正附加仍由供应商 models_1m 支持表决定。 */
export function canAttach1M(slot: ModelSlot): boolean {
  return slot === 'opus' || slot === 'sonnet' || slot === 'fable';
}

/** 从模型列表选中一个模型后的落值：允许附加且命中 1M 表时加 [1m] 后缀（已带后缀则幂等不重复）。
 *  纯函数，便于矩阵单测（对齐 Go 版 presets.ApplyModelSelection）。 */
export function applyModelSelection(slot: ModelSlot, modelId: string, supports1M: boolean): string {
  if (!canAttach1M(slot) || !supports1M) return modelId;
  return modelId.replace(/\[1m\]$/, '') + '[1m]';
}

const MODELS_TIMEOUT_MS = 10_000;

/** 模型列表响应上限（对齐 Go 版 LimitReader 的 1 MiB；防御异常/恶意端点的大响应）。 */
const MAX_RESPONSE_BYTES = 1 << 20;

/** 流式读取响应体为 UTF-8 文本；累计超过 MAX_RESPONSE_BYTES 时取消响应并明确报错。
 *  不检查 Content-Length（chunked 响应可以不带它），必须逐块累计。 */
async function readLimitedText(res: Response, limit: number): Promise<string> {
  if (!res.body) return '';
  const reader = res.body.getReader();
  const decoder = new TextDecoder('utf-8');
  let text = '';
  let total = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > limit) {
      await reader.cancel();
      throw new Error('响应超过 1 MiB，已拒绝');
    }
    text += decoder.decode(value, { stream: true });
  }
  return text + decoder.decode();
}

/** 拉取模型列表。baseUrl 是 Anthropic 兼容端点，token 是认证凭据（Bearer token 或 API key），
 *  auth 决定携带哪个认证头——与编辑表单的 auth 选择一致，也即 Claude Code 实际调用时使用的头。
 *  endpoint 为空时推导 {base}/v1/models。3xx 重定向一律拒绝：模型列表端点不应重定向，
 *  且跨主机重定向会泄露认证头（浏览器/Go 的敏感头剥离列表都不含 x-api-key）。 */
export async function fetchModels(baseUrl: string, token: string, auth: AuthMode, endpoint?: string): Promise<ModelInfo[]> {
  const url = endpoint?.trim() || `${baseUrl.replace(/\/+$/, '')}/v1/models`;
  const headers: Record<string, string> = {
    'User-Agent': 'ccx',
    'anthropic-version': '2023-06-01',
  };
  if (auth === 'API_KEY') headers['x-api-key'] = token;
  else headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(url, {
    method: 'GET',
    headers,
    redirect: 'manual',
    signal: AbortSignal.timeout(MODELS_TIMEOUT_MS),
  });
  if (res.status >= 300 && res.status < 400) {
    throw new Error(`模型列表端点发生重定向，已拒绝：${res.headers.get('location') ?? '未知地址'}`);
  }
  const body = await readLimitedText(res, MAX_RESPONSE_BYTES);
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
