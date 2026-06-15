/**
 * 三级菜单：主菜单 ↔ 动作菜单 ↔ 编辑表单（M4）。
 *
 * 主菜单：列表 + 排序（Shift+↑↓/PgUp/PgDn）+ 记忆选中 + 新增 + 语言切换 + 退出。
 * 动作菜单：本次启用 / 设为默认（绿条 toast）/ 编辑 / 删除（二次确认）/ 返回。
 * 编辑表单见 ui/edit.ts（含密钥明文切换）。
 */
import { launchSession } from '../actions.js';
import { checkProfile } from '../check.js';
import { getProviderEnvMap, getProviderState, isOfficial, reconcileCurrent, saveStore, type Provider, type Store, type StorePaths } from '../config/store.js';
import type { Preset } from '../config/types.js';
import { persistDefaultEnv, setDefault, type DefaultScope } from '../env/default.js';
import { getLang, providerDisplayName, setLang, T } from '../i18n/index.js';
import { currentTerminalLine, hostOf } from '../runtime-info.js';
import { banner as updateBanner, maybeRefresh, MODE_NOTIFY, upgradeCommand } from '../update/update.js';
import { paint } from '../utils/ansi.js';
import { displayWidth, padDisplay } from '../utils/display.js';
import { editForm } from './edit.js';
import { noteSuffix, stateLabel } from './format.js';
import { confirmKey, selectMenu } from './select.js';

/** 一级 · 主菜单。布局：[profiles…] '' 新增 语言 '' 退出。 */
export async function openMenu(
  paths: StorePaths,
  store: Store,
  scope: DefaultScope,
  version: string,
  catalog: Preset[],
): Promise<void> {
  let sel = 0;
  let refreshed = false;
  let flash: string | undefined;
  let warnFlash: string | undefined;
  for (;;) {
    const n = store.providers.length;
    // 更新检查（仅 notify 模式）：首轮触发一次后台刷新；横幅永远读缓存（瞬时、不阻塞）。
    const notices = [currentTerminalLine(store)];
    if (needsFirstRunHint(store)) notices.push(T('menu.firstRunHint'));
    if (warnFlash) notices.push(warnFlash);
    if (store.update === MODE_NOTIFY) {
      if (!refreshed) {
        maybeRefresh(paths.dir);
        refreshed = true;
      }
      const latest = updateBanner(paths.dir, version);
      if (latest) notices.push(T('menu.updateAvailable', latest, upgradeCommand()));
    }
    const updLabel = store.update === MODE_NOTIFY ? T('menu.updateNotify') : T('menu.updateOff');
    const buildItems = (): string[] => {
      const labels = profileRows(store.providers, store.current);
      return [...labels, '', T('menu.newProfile'), T('menu.language'), updLabel, '', T('menu.exit')];
    };
    const onMove = (from: number, to: number): string[] => {
      const ps = store.providers;
      const a = ps[from];
      const b = ps[to];
      if (a && b) {
        ps[from] = b;
        ps[to] = a;
        saveStore(paths, store);
      }
      return buildItems();
    };

    const defaultName = defaultDisplayName(store);
    let shortcut = '';
    sel = await selectMenu({
      title: T('menu.mainTitle', version, defaultName),
      notice: notices.join('\n'),
      ...(flash ? { status: flash } : {}),
      items: buildItems(),
      colors: { [n + 1]: 'yellow' },
      start: sel,
      movableCount: n,
      onMove,
      onKey: (r: string, idx: number): number => {
        if (idx >= n) return -1;
        if (r === 'e' || r === 's' || r === 'd') {
          shortcut = r;
          return idx;
        }
        return -1;
      },
      hint: T('menu.mainHint'),
      noNumber: true,
    });
    flash = undefined;
    warnFlash = undefined;

    if (shortcut && sel >= 0 && sel < n) {
      const target = store.providers[sel];
      if (!target) continue;
      if (shortcut === 'e') {
        const old = target.name;
        if (await editForm(target, store, catalog)) {
          ({ warn: warnFlash, toast: flash } = saveEditedProfile(paths, store, target, old, scope));
        }
      } else if (shortcut === 's') {
        launchSession(target);
      } else if (shortcut === 'd') {
        ({ warn: warnFlash, toast: flash } = applyDefault(paths, store, target, scope));
      }
      continue;
    }

    if (sel < 0 || sel === n + 5) return; // 退出 / Esc / q
    if (sel === n + 1) {
      // 新增配置
      const prov: Provider = { name: '', env: {} };
      if (await editForm(prov, store, catalog)) {
        store.providers.push(prov);
        saveStore(paths, store);
        sel = store.providers.length - 1; // 光标落到新配置
      }
    } else if (sel === n + 2) {
      // 语言切换：即时切并写回 store.lang
      const next = getLang() === 'zh' ? 'en' : 'zh';
      setLang(next);
      store.lang = next;
      saveStore(paths, store);
    } else if (sel === n + 3) {
      // 更新检查开关：关闭 <-> 提醒（关闭=删字段，与 Go 的 omitempty 对齐）
      if (store.update === MODE_NOTIFY) delete store.update;
      else store.update = MODE_NOTIFY;
      saveStore(paths, store);
    } else if (sel < n) {
      const target = store.providers[sel];
      if (target) {
        if (!isOfficial(target) && getProviderState(target).key === 'noKey') {
          // #9：无密钥的第三方配置，Enter 直达编辑并聚焦密钥行（铺平首次成功路径）。
          const old = target.name;
          if (await editForm(target, store, catalog, true)) {
            ({ warn: warnFlash, toast: flash } = saveEditedProfile(paths, store, target, old, scope));
          }
        } else {
          await actionMenu(paths, store, target, scope, catalog);
        }
      }
      if (sel >= store.providers.length) sel = Math.max(0, store.providers.length - 1); // 删除后夹取
    }
  }
}

/** 二级 · 动作菜单（循环停留；返回/删除已确认才回一级）。 */
async function actionMenu(
  paths: StorePaths,
  store: Store,
  p: Provider,
  scope: DefaultScope,
  catalog: Preset[],
): Promise<void> {
  let sel = 0;
  let flash: string | undefined;
  let warnFlash: string | undefined; // 黄字警告（如缺密钥），走 notice 与绿色 status 区分
  for (;;) {
    const dft = p.name === store.current ? T('menu.default') : '';
    const title = `${T('action.titlePrefix')}${providerDisplayName(p)}${dft}${noteSuffix(p)}    [${stateLabel(p)}]`;
    const items = [T('action.session'), T('action.setDefault'), T('action.check'), T('action.edit'), T('action.delete'), T('action.back')];

    sel = await selectMenu({
      title,
      items,
      start: sel,
      ...(warnFlash ? { notice: warnFlash } : {}),
      ...(flash ? { status: flash } : {}),
      hint: T('action.hint'),
      noNumber: true,
    });
    flash = undefined;
    warnFlash = undefined;

    if (sel === 0) {
      launchSession(p);
    } else if (sel === 1) {
      ({ warn: warnFlash, toast: flash } = applyDefault(paths, store, p, scope));
    } else if (sel === 2) {
      const result = await checkProfile(p);
      if (result.ok) flash = result.message;
      else warnFlash = result.message;
    } else if (sel === 3) {
      const old = p.name;
      if (await editForm(p, store, catalog)) {
        ({ warn: warnFlash, toast: flash } = saveEditedProfile(paths, store, p, old, scope));
      }
    } else if (sel === 4) {
      if (isOfficial(p)) console.log(`  ${T('action.deleteOfficialWarn')}`);
      if (await confirmKey(T('action.deleteConfirm', providerDisplayName(p)))) {
        store.providers = store.providers.filter((x) => x !== p);
        reconcileCurrent(store);
        saveStore(paths, store);
        return;
      }
    } else {
      return; // 返回 / q / Esc
    }
  }
}

function defaultDisplayName(store: Store): string {
  if (!store.current) return '—';
  return providerDisplayName(store.providers.find((p) => p.name === store.current) ?? { name: store.current, env: {} });
}

function saveEditedProfile(paths: StorePaths, store: Store, p: Provider, oldName: string, scope: DefaultScope): { warn?: string; toast?: string } {
  const wasDefault = store.current === oldName;
  if (wasDefault) store.current = p.name; // 改了名/供应商时同步默认指向
  saveStore(paths, store);
  if (wasDefault) return syncDefaultEnv(p, scope);
  return {};
}

function defaultWarning(p: Provider): string {
  return getProviderState(p).key === 'noKey' ? T('default.noKey', providerDisplayName(p)) : '';
}

function defaultResultMessage(warn: string, name: string, r: ReturnType<typeof setDefault>): { warn: string; toast: string } {
  if (r.dryRun) return { warn, toast: `${T('default.done', name)}  ${T('default.dryRun')}` };
  if (r.windows && !r.windows.ok) return { warn, toast: T('default.failed', r.windows.error ?? '') };
  if (r.unix?.unsupported) return { warn, toast: T('default.fishUnsupported') };
  return { warn, toast: T('default.done', name) };
}

function syncDefaultEnv(p: Provider, scope: DefaultScope): { warn: string; toast: string } {
  const name = providerDisplayName(p);
  return defaultResultMessage(defaultWarning(p), name, persistDefaultEnv(p, scope));
}

/**
 * 设为默认，返回 { warn, toast }：warn 为黄字警告（缺密钥），toast 为绿色结果。
 * 分开返回让调用方各自上色，避免警告被染成「成功」绿。
 */
function applyDefault(paths: StorePaths, store: Store, p: Provider, scope: DefaultScope): { warn: string; toast: string } {
  const name = providerDisplayName(p);
  return defaultResultMessage(defaultWarning(p), name, setDefault(paths, store, p, scope));
}

const ROW_NAME_W = 13; // 名字列宽（显示宽度，CJK 计 2）；状态/备注列宽按内容动态算

// profileRows 把所有配置格式化成定宽分栏的菜单行：名字为主信息（默认项加粗），
// 状态/备注/host 一律 dim 退为次要信息，让备注、host 各自对齐成竖列。
// 状态、备注列宽按当前配置实际内容动态取最大值——自动兜住任何语言/文案。
// TUI 主菜单与 xx --list 共用此函数，保证两处呈现一致。与 Go 版 menu.go ProfileRows 对齐。
export function profileRows(providers: Provider[], current: string): string[] {
  let stateW = 0;
  let noteW = 0;
  for (const p of providers) {
    stateW = Math.max(stateW, displayWidth(stateLabel(p)));
    noteW = Math.max(noteW, displayWidth(p.note ?? ''));
  }
  stateW += 2; // 状态列尾留出与备注列的间距（最宽行否则会与备注贴死）
  if (noteW > 0) noteW += 2; // 有备注时留出与 host 的列间距
  return providers.map((p) => profileRow(p, p.name === current, stateW, noteW));
}

// profileRow 组成一行：定宽分栏 + 次要信息 dim + 默认项加粗。
// 加粗用 1m/22m（reset 不动颜色），不会破坏 selectMenu 选中行的外层绿色。
function profileRow(p: Provider, isDefault: boolean, stateW: number, noteW: number): string {
  let name = padDisplay(providerDisplayName(p), ROW_NAME_W);
  let marker = '  ';
  if (isDefault) {
    marker = paint('● ', 'bold');
    name = paint(name, 'bold');
  }
  const state = paint(` · ${padDisplay(stateLabel(p), stateW)}`, 'dim');
  // 空备注=定宽空白，撑出 host 前的列间距
  const note = (p.note ?? '').trim()
    ? paint(padDisplay(p.note ?? '', noteW), 'dim')
    : padDisplay('', noteW);
  return `${marker}${name}${state}${note}${hostCol(p)}`;
}

// hostCol 返回行尾 dim 的 host（无 base 返回空）。超宽时由 selectMenu 的 ANSI-aware 截断裁掉。
function hostCol(p: Provider): string {
  const base = (getProviderEnvMap(p).ANTHROPIC_BASE_URL ?? '').trim();
  if (!base) return '';
  return paint(hostOf(base), 'dim');
}

function needsFirstRunHint(store: Store): boolean {
  let hasThirdParty = false;
  for (const p of store.providers) {
    if (isOfficial(p)) continue;
    hasThirdParty = true;
    if (getProviderState(p).key !== 'noKey') return false;
  }
  return hasThirdParty;
}
