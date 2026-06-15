/**
 * 三级 · 编辑表单（对齐现版 Edit-Form）：一屏显示全部字段，选序号改单项。
 *
 * 新需求（plan §7）：密钥行默认掩码 `********`，提供「👁 显示/隐藏密钥明文」开关——
 * 仅影响本表单的**显示**，不改数据、不持久化；默认隐藏防肩窥。输入态（readValue secret）另算。
 */
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
  effort: string;
  disableTraffic: string;
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
  // rows 布局固定：provider,note,base,auth,key,…（密钥行索引为 4）。
  let start = focusKey ? 4 : 0;

  for (;;) {
    const v = (x: string): string => (x === '' ? T('empty.paren') : x);
    const keyDisp = W.token === '' ? T('empty.paren') : showSecret ? W.token : '********';
    const rows: Array<{ action: string; label: string }> = [
      { action: 'provider', label: `${T('edit.field.provider')}: ${v(W.name)}` },
      { action: 'note', label: `${T('edit.field.note')}: ${v(W.note)}` },
      { action: 'base', label: `${T('edit.field.base')}: ${v(W.base)}` },
      { action: 'auth', label: `${T('edit.field.auth')}: ${W.auth}` },
      { action: 'key', label: `${T('edit.field.key')}: ${keyDisp}` },
      { action: 'opus', label: `${T('edit.field.opus')}: ${v(W.opus)}` },
      { action: 'sonnet', label: `${T('edit.field.sonnet')}: ${v(W.sonnet)}` },
      { action: 'haiku', label: `${T('edit.field.haiku')}: ${v(W.haiku)}` },
      { action: 'effort', label: `${T('edit.field.effort')}: ${v(W.effort)}` },
      { action: 'disableTraffic', label: `${T('edit.field.disableTraffic')}: ${v(W.disableTraffic)}` },
      { action: 'sep', label: '' },
      { action: 'toggle', label: showSecret ? T('edit.toggleSecretHide') : T('edit.toggleSecretShow') },
      { action: 'sep', label: '' },
      { action: 'save', label: T('edit.save') },
      { action: 'discard', label: T('edit.discard') },
    ];

    const sel = await selectMenu({ title: T('edit.title'), items: rows.map((r) => r.label), start, hint: T('edit.hint'), noNumber: true });
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
        const r = await readValue(T('edit.field.opus').trim(), W.opus);
        if (r.changed) W.opus = r.value;
        break;
      }
      case 'sonnet': {
        const r = await readValue(T('edit.field.sonnet').trim(), W.sonnet);
        if (r.changed) W.sonnet = r.value;
        break;
      }
      case 'haiku': {
        const r = await readValue(T('edit.field.haiku').trim(), W.haiku);
        if (r.changed) W.haiku = r.value;
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
