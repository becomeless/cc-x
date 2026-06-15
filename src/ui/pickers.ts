/**
 * 编辑表单里的各 picker（供应商 / API 地址 / 认证字段 / effort），对齐现版 Pick-*。
 */
import { getProviderEnvMap, type Store } from '../config/store.js';
import type { Preset } from '../config/types.js';
import { T } from '../i18n/index.js';
import { padDisplay } from '../utils/display.js';
import { selectMenu } from './select.js';
import { readText } from './text.js';

/** 选供应商：从目录选一个 / 自定义手填名 / 不改。 */
export async function pickProvider(catalog: Preset[], current: string): Promise<Preset | 'custom' | null> {
  const names = catalog.map((p) => p.name);
  const items = [...names, T('pick.provider.custom'), T('pick.noChange')];
  const sel = await selectMenu({
    title: T('pick.provider.title', current || T('pick.provider.none')),
    items,
    hint: T('pick.hint'),
    noNumber: true,
  });
  if (sel < 0 || sel === items.length - 1) return null;
  if (sel === names.length) return 'custom';
  return catalog[sel] ?? null;
}

/** 供应商有多个地址时让用户选一个；只有一个直接用，无地址保持原值。 */
export async function pickProviderUrl(preset: Preset, current: string): Promise<string> {
  const urls = preset.urls;
  if (urls.length === 0) return current;
  if (urls.length === 1) return urls[0]?.url ?? current;
  const labels = urls.map((u) => `${padDisplay(u.label, 12)} ${u.url || T('empty.paren')}`);
  const items = [...labels, T('pick.noChange')];
  const sel = await selectMenu({ title: T('pick.providerUrl.title', preset.name), items, hint: T('pick.hint'), noNumber: true });
  if (sel < 0 || sel === items.length - 1) return current;
  return urls[sel]?.url ?? current;
}

/** 选 API 地址：目录所有 url + 已有配置用过的 url + 手动输入 + 不修改。 */
export async function pickBaseUrl(current: string, store: Store, catalog: Preset[]): Promise<string> {
  const entries: Array<{ label: string; url: string }> = [];
  const seen = new Set<string>();
  for (const p of catalog) {
    for (const u of p.urls) {
      const tag = p.urls.length > 1 ? `${p.name}/${u.label}` : p.name;
      entries.push({ label: `${padDisplay(tag, 20)} ${u.url || T('empty.paren')}`, url: u.url });
      seen.add(u.url);
    }
  }
  for (const prov of store.providers) {
    const u = getProviderEnvMap(prov).ANTHROPIC_BASE_URL;
    if (u && u.trim() && !seen.has(u)) {
      seen.add(u);
      entries.push({ label: `${padDisplay(T('pick.base.existing', prov.name), 20)} ${u}`, url: u });
    }
  }
  const items = [...entries.map((e) => e.label), T('pick.manual'), T('pick.noChange')];
  const sel = await selectMenu({ title: T('pick.base.title', current || T('empty.paren')), items, hint: T('pick.hint'), noNumber: true });
  if (sel < 0 || sel === items.length - 1) return current;
  if (sel < entries.length) return entries[sel]?.url ?? current;
  const v = await readText(`  ${T('pick.base.manualInput')}`);
  if (v === undefined || v === '') return current;
  if (v === '-') return '';
  return v.trim();
}

/** 选认证字段：AUTH_TOKEN / API_KEY / 不改。 */
export async function pickAuth(current: 'AUTH_TOKEN' | 'API_KEY'): Promise<'AUTH_TOKEN' | 'API_KEY'> {
  const items = [T('pick.auth.token'), T('pick.auth.apikey'), T('pick.noChange')];
  const sel = await selectMenu({ title: T('pick.auth.title', current), items, hint: T('pick.hint'), noNumber: true });
  if (sel === 0) return 'AUTH_TOKEN';
  if (sel === 1) return 'API_KEY';
  return current;
}

const EFFORT_OPTS = ['low', 'medium', 'high', 'xhigh', 'max', 'auto'];

/** 选是否禁用非核心流量：是（"1"）/ 否（""）/ 不改。 */
export async function pickDisableTraffic(current: string): Promise<string> {
  const items = [T('pick.disableTraffic.yes'), T('pick.disableTraffic.no'), T('pick.noChange')];
  const sel = await selectMenu({
    title: T('pick.disableTraffic.title', current || T('empty.paren')),
    items,
    hint: T('pick.hint'),
    noNumber: true,
  });
  if (sel < 0 || sel === 2) return current;
  return sel === 0 ? '1' : '';
}

/** 选 effort 思考档：low…auto / 留空 / 不改。 */
export async function pickEffort(current: string): Promise<string> {
  const items = [...EFFORT_OPTS, T('pick.effort.empty'), T('pick.noChange')];
  const sel = await selectMenu({
    title: T('pick.effort.title', current || T('empty.paren')),
    items,
    hint: T('pick.effort.hint'),
    noNumber: true,
  });
  if (sel < 0 || sel === items.length - 1) return current;
  if (sel === EFFORT_OPTS.length) return '';
  return EFFORT_OPTS[sel] ?? current;
}
