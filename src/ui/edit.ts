/**
 * 三级 · 编辑表单（对齐现版 Edit-Form）：一屏显示全部字段，选序号改单项。
 *
 * 新需求（plan §7）：密钥行默认掩码 `********`，提供「👁 显示/隐藏密钥明文」开关——
 * 仅影响本表单的**显示**，不改数据、不持久化；默认隐藏防肩窥。输入态（readValue secret）另算。
 */
import {
  applyModelSelection,
  canAttach1M,
  fetchModels,
  supports1M,
  type ModelInfo,
  type ModelSlot,
} from '../config/models.js';
import {
  buildProviderEnv,
  getProviderEnvMap,
  reconcileBuiltin,
  resolveUniqueName,
  type Provider,
  type Store,
} from '../config/store.js';
import type { Preset } from '../config/types.js';
import { T } from '../i18n/index.js';
import { displayWidth, padDisplay } from '../utils/display.js';
import { pickAuth, pickBaseUrl, pickDisableTraffic, pickEffort, pickProvider, pickProviderUrl } from './pickers.js';
import { selectMenu } from './select.js';
import { readText, readValue } from './text.js';

interface WorkCopy {
  name: string;
  note: string;
  base: string;
  auth: 'AUTH_TOKEN' | 'API_KEY';
  token: string;
  opus: string;
  sonnet: string;
  haiku: string;
  fable: string;
  subagent: string;
  effort: string;
  disableTraffic: string;
}

function findPreset(catalog: Preset[], name: string): Preset | undefined {
  return catalog.find((p) => p.name === name);
}

/** 子代理行为空时显示「默认」（不强制覆盖；未另行指定时继承主模型），不显示 (空)。 */
function subagentLabel(v: string): string {
  return v.trim() === '' ? T('edit.default') : v;
}

/** 按 BASE_URL 前缀在目录里找供应商；配置名与预设名不同（如「DeepSeek 2」）也能命中。 */
function findPresetByBase(catalog: Preset[], baseUrl: string): Preset | undefined {
  for (const p of catalog) {
    for (const u of p.urls) {
      if (u.url !== '' && baseUrl.startsWith(u.url)) return p;
    }
    if (p.modelsApiMap) {
      for (const k of Object.keys(p.modelsApiMap)) {
        if (baseUrl.startsWith(k)) return p;
      }
    }
  }
  return undefined;
}

/** 定位当前表单的供应商预设：先按 base URL（同名不同配置也能命中），再按配置名兜底。 */
function catalogPreset(catalog: Preset[], W: WorkCopy): Preset | undefined {
  return findPresetByBase(catalog, W.base) ?? findPreset(catalog, W.name);
}

/** 解析模型列表端点：models_api_map 按 base 前缀匹配 > models_api > undefined（推导）。 */
function resolveModelsEndpoint(pp: Preset | undefined, baseUrl: string): string | undefined {
  if (!pp) return undefined;
  if (pp.modelsApiMap) {
    for (const [base, ep] of Object.entries(pp.modelsApiMap)) {
      if (baseUrl.startsWith(base)) return ep;
    }
  }
  return pp.modelsApi;
}

/** 构建模型列表菜单条目与 1M 标记（仅 opus/sonnet 档命中供应商 1M 表才标 [1M]）。对齐 Go 版 buildModelItems。
 *  有 display_name 且与 ID 不同时显示「友好名 (实际ID)」——主标签可读，括号里是选中后真正填入的值。 */
function buildModelItems(models: ModelInfo[], pp: Preset | undefined, slot: ModelSlot): { items: string[]; is1M: boolean[] } {
  const is1M = models.map((m) => canAttach1M(slot) && supports1M(pp, m.id));
  const items = models.map((m, i) => {
    let label = m.id;
    if (m.displayName && m.displayName !== m.id) {
      label = `${m.displayName} (${m.id})`;
    }
    if (is1M[i]) label += '  [1M]';
    return label;
  });
  return { items, is1M };
}

/** 从已拉取的模型列表选一个；opus/sonnet 档命中 1M 表自动附加 [1m] 后缀（想用 200K 可在表单行手动删）。
 *  haiku/fable 档永不附加。取消返回 null。 */
async function pickFromList(
  models: ModelInfo[],
  items: string[],
  is1M: boolean[],
  slot: ModelSlot,
  title: string,
  hint: string,
): Promise<string | null> {
  const sel = await selectMenu({ title, items, start: 0, hint, noNumber: true });
  if (sel < 0) return null;
  const picked = models[sel];
  if (!picked) return null;
  return applyModelSelection(slot, picked.id, is1M[sel] ?? false);
}

/**
 * 从供应商 API 拉取模型列表并让用户选一个。返回选中模型 ID；用户取消返回 null；失败 throw
 * （由表单 Status 展示，错误不再被清屏吞掉）。
 */
async function fetchAndPickModel(W: WorkCopy, catalog: Preset[], slot: ModelSlot, title: string): Promise<string | null> {
  if (W.base.trim() === '' || W.token.trim() === '') {
    throw new Error(T('models.needBaseKey'));
  }
  const pp = catalogPreset(catalog, W);
  console.log(`  ${T('models.fetching')}`);
  let models: ModelInfo[];
  try {
    models = await fetchModels(W.base, W.token, W.auth, resolveModelsEndpoint(pp, W.base));
  } catch (err) {
    throw err instanceof Error ? err : new Error(String(err));
  }
  const { items, is1M } = buildModelItems(models, pp, slot);
  return pickFromList(models, items, is1M, slot, title, T(canAttach1M(slot) ? 'models.hint' : 'models.hintNoSuffix'));
}

/** 档位行的编辑入口：手动输入 / 从模型列表选择。失败 throw（由表单 Status 展示）。 */
async function pickSlotModel(
  W: WorkCopy,
  catalog: Preset[],
  slot: ModelSlot,
  label: string,
  current: string,
): Promise<{ changed: boolean; value: string }> {
  const cur = current === '' ? T('empty.paren') : current;
  const opts = [T('models.pickManual'), T('models.pickFromList')];
  const sel = await selectMenu({ title: `${label}（当前：${cur}）`, items: opts, start: 0, hint: T('pick.hint'), noNumber: true });
  if (sel < 0) return { changed: false, value: current };
  if (sel === 0) {
    return readValue(label, current);
  }
  const model = await fetchAndPickModel(W, catalog, slot, T('models.pickTitle'));
  if (model === null) return { changed: false, value: current };
  return { changed: true, value: model };
}

function fromProvider(p: Provider): WorkCopy {
  const m = getProviderEnvMap(p);
  const usesApiKey = Boolean(m.ANTHROPIC_API_KEY && m.ANTHROPIC_API_KEY.trim());
  return {
    name: p.name,
    note: p.note ?? '',
    base: m.ANTHROPIC_BASE_URL ?? '',
    auth: usesApiKey ? 'API_KEY' : 'AUTH_TOKEN',
    token: (usesApiKey ? m.ANTHROPIC_API_KEY : m.ANTHROPIC_AUTH_TOKEN) ?? '',
    opus: m.ANTHROPIC_DEFAULT_OPUS_MODEL ?? '',
    sonnet: m.ANTHROPIC_DEFAULT_SONNET_MODEL ?? '',
    haiku: m.ANTHROPIC_DEFAULT_HAIKU_MODEL ?? '',
    fable: m.ANTHROPIC_DEFAULT_FABLE_MODEL ?? '',
    subagent: m.CLAUDE_CODE_SUBAGENT_MODEL ?? '',
    effort: m.CLAUDE_CODE_EFFORT_LEVEL ?? '',
    disableTraffic: m.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC ?? '',
  };
}

/**
 * 编辑 `prov`（就地修改）；保存返回 true，放弃返回 false。
 * focusKey=true 时初始光标落在密钥行（#9：无 key 配置 Enter 直达填密钥的最短路径）。
 */
export async function editForm(prov: Provider, store: Store, catalog: Preset[], focusKey = false): Promise<boolean> {
  const W = fromProvider(prov);
  let showSecret = false;
  let status = ''; // 表单顶部绿色提示条：最近一次模型操作的结果/错误（清屏不会吞掉它）
  // rows 布局固定：provider,note,base,auth,key,…（密钥行索引为 4）。
  let start = focusKey ? 4 : 0;

  for (;;) {
    const v = (x: string): string => (x === '' ? T('empty.paren') : x);
    const keyDisp = W.token === '' ? T('empty.paren') : showSecret ? W.token : '********';
    // 12 个字段行（顺序即行序，密钥行索引为 4）。标签按显示宽度统一补齐后再拼冒号——
    // 中英混排 + 全角/半角字符时手写尾部空格数不可靠（→ 宽度还随终端字体变化），对齐在渲染时计算。
    const fieldRows: Array<{ action: string; label: string; value: string }> = [
      { action: 'provider', label: T('edit.field.provider'), value: v(W.name) },
      { action: 'note', label: T('edit.field.note'), value: v(W.note) },
      { action: 'base', label: T('edit.field.base'), value: v(W.base) },
      { action: 'auth', label: T('edit.field.auth'), value: W.auth },
      { action: 'key', label: T('edit.field.key'), value: keyDisp },
      { action: 'opus', label: T('edit.field.opus'), value: v(W.opus) },
      { action: 'sonnet', label: T('edit.field.sonnet'), value: v(W.sonnet) },
      { action: 'haiku', label: T('edit.field.haiku'), value: v(W.haiku) },
      { action: 'fable', label: T('edit.field.fable'), value: v(W.fable) },
      { action: 'subagent', label: T('edit.field.subagent'), value: subagentLabel(W.subagent) },
      { action: 'effort', label: T('edit.field.effort'), value: v(W.effort) },
      { action: 'disableTraffic', label: T('edit.field.disableTraffic'), value: v(W.disableTraffic) },
    ];
    const labelW = fieldRows.reduce((m, f) => Math.max(m, displayWidth(f.label)), 0);
    const rows: Array<{ action: string; label: string }> = [
      ...fieldRows.map((f) => ({ action: f.action, label: `${padDisplay(f.label, labelW)}: ${f.value}` })),
      { action: 'sep', label: '' },
      { action: 'toggle', label: showSecret ? T('edit.toggleSecretHide') : T('edit.toggleSecretShow') },
      { action: 'sep', label: '' },
      { action: 'save', label: T('edit.save') },
      { action: 'discard', label: T('edit.discard') },
    ];

    const sel = await selectMenu({ title: T('edit.title'), items: rows.map((r) => r.label), start, hint: T('edit.hint'), status, noNumber: true });
    if (sel < 0) return false; // Esc / q = 放弃
    start = sel;

    switch (rows[sel]?.action) {
      case 'provider': {
        const pp = await pickProvider(catalog, W.name);
        if (pp === 'custom') {
          const name = await readText(`  ${T('edit.customName')}`);
          if (name && name.trim()) W.name = name.trim();
        } else if (pp) {
          W.name = pp.name;
          W.auth = pp.auth;
          W.base = await pickProviderUrl(pp, W.base);
          if (pp.models.opus) W.opus = pp.models.opus;
          if (pp.models.sonnet) W.sonnet = pp.models.sonnet;
          if (pp.models.haiku) W.haiku = pp.models.haiku;
          if (pp.models.fable) W.fable = pp.models.fable;
          if (pp.effort) W.effort = pp.effort;
          if (pp.env && Object.prototype.hasOwnProperty.call(pp.env, 'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC')) {
            W.disableTraffic = pp.env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC ?? '';
          }
        }
        break;
      }
      case 'note': {
        const note = await readText(`  ${T('edit.noteInput')}`);
        if (note === '-') W.note = '';
        else if (note && note.trim()) W.note = note.trim();
        break;
      }
      case 'base':
        W.base = await pickBaseUrl(W.base, store, catalog);
        break;
      case 'auth':
        W.auth = await pickAuth(W.auth);
        break;
      case 'key': {
        const r = await readValue(T('edit.field.key').trim(), W.token, true);
        if (r.changed) W.token = r.value;
        break;
      }
      case 'opus': {
        try {
          const r = await pickSlotModel(W, catalog, 'opus', T('edit.field.opus').trim(), W.opus);
          if (r.changed) {
            W.opus = r.value;
            status = '';
          }
        } catch (err) {
          status = T('models.fail', err instanceof Error ? err.message : String(err));
        }
        break;
      }
      case 'sonnet': {
        try {
          const r = await pickSlotModel(W, catalog, 'sonnet', T('edit.field.sonnet').trim(), W.sonnet);
          if (r.changed) {
            W.sonnet = r.value;
            status = '';
          }
        } catch (err) {
          status = T('models.fail', err instanceof Error ? err.message : String(err));
        }
        break;
      }
      case 'haiku': {
        try {
          const r = await pickSlotModel(W, catalog, 'haiku', T('edit.field.haiku').trim(), W.haiku);
          if (r.changed) {
            W.haiku = r.value;
            status = '';
          }
        } catch (err) {
          status = T('models.fail', err instanceof Error ? err.message : String(err));
        }
        break;
      }
      case 'fable': {
        try {
          const r = await pickSlotModel(W, catalog, 'fable', T('edit.field.fable').trim(), W.fable);
          if (r.changed) {
            W.fable = r.value;
            status = '';
          }
        } catch (err) {
          status = T('models.fail', err instanceof Error ? err.message : String(err));
        }
        break;
      }
      case 'subagent': {
        const r = await readValue(T('edit.field.subagent').trim(), W.subagent);
        if (r.changed) W.subagent = r.value;
        break;
      }
      case 'effort':
        W.effort = await pickEffort(W.effort);
        break;
      case 'disableTraffic':
        W.disableTraffic = await pickDisableTraffic(W.disableTraffic);
        break;
      case 'toggle':
        showSecret = !showSecret; // 仅切换显示，不改数据、不持久化
        break;
      case 'save': {
        if (W.name.trim() === '') {
          console.log(`  ${T('edit.nameEmpty')}`);
          break;
        }
        const fields: Record<string, string> = {
          ANTHROPIC_BASE_URL: W.base,
          ANTHROPIC_DEFAULT_OPUS_MODEL: W.opus,
          ANTHROPIC_DEFAULT_SONNET_MODEL: W.sonnet,
          ANTHROPIC_DEFAULT_HAIKU_MODEL: W.haiku,
          ANTHROPIC_DEFAULT_FABLE_MODEL: W.fable,
          CLAUDE_CODE_SUBAGENT_MODEL: W.subagent,
          CLAUDE_CODE_EFFORT_LEVEL: W.effort,
          CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: W.disableTraffic,
        };
        if (W.auth === 'API_KEY') fields.ANTHROPIC_API_KEY = W.token;
        else fields.ANTHROPIC_AUTH_TOKEN = W.token;
        prov.name = resolveUniqueName(store, W.name.trim(), prov);
        prov.env = buildProviderEnv(fields);
        prov.note = W.note;
        reconcileBuiltin(prov); // [P1] 官方档被配成第三方后清掉 builtin 身份
        return true;
      }
      case 'discard':
        return false;
      default:
        break; // sep
    }
  }
}
