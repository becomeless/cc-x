/**
 * 三级菜单：主菜单 ↔ 动作菜单 ↔ 编辑表单（M4）。
 *
 * 主菜单：列表 + 排序（Shift+↑↓/PgUp/PgDn）+ 记忆选中 + 新增 + 语言切换 + 退出。
 * 动作菜单：本次启用 / 设为默认（绿条 toast）/ 编辑 / 删除（二次确认）/ 返回。
 * 编辑表单见 ui/edit.ts（含密钥明文切换）。
 */
import { launchSession } from '../actions.js';
import { markOnboardingDone } from '../claudecfg.js';
import { checkProfile } from '../check.js';
import { getProviderState, isOfficial, reconcileCurrent, saveStore, type Provider, type Store, type StorePaths } from '../config/store.js';
import type { Preset } from '../config/types.js';
import { persistDefaultEnv, setDefault, type DefaultScope } from '../env/default.js';
import { getLang, providerDisplayName, setLang, T } from '../i18n/index.js';
import { banner as updateBanner, maybeRefresh, MODE_NOTIFY, upgradeCommand } from '../update/update.js';
import { paint } from '../utils/ansi.js';
import { displayWidth, padDisplay } from '../utils/display.js';
import { editForm } from './edit.js';
import { noteSuffix, stateLabel } from './format.js';
import { confirmKey, selectMenu } from './select.js';

/** 一级 · 主菜单。布局：[profiles…] '' 新增 语言 更新检查 免登录 '' 退出。 */
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
  // 更新横幅（仅 notify 模式）：缓存文件只在首轮读一次（每按键都读是浪费），存进来循环内复用。
  let bannerLatest: string | undefined;
  for (;;) {
    const n = store.providers.length;
    const notices: string[] = [];
    if (needsFirstRunHint(store)) notices.push(T('menu.firstRunHint'));
    if (warnFlash) notices.push(warnFlash);
    if (store.update === MODE_NOTIFY) {
      if (!refreshed) {
        maybeRefresh(paths.dir);
        refreshed = true;
        bannerLatest = updateBanner(paths.dir, version);
      }
      if (bannerLatest) notices.push(T('menu.updateAvailable', bannerLatest, upgradeCommand()));
    }
    const updLabel = store.update === MODE_NOTIFY ? T('menu.updateNotify') : T('menu.updateOff');
    const buildItems = (): string[] => {
      const labels = profileRows(store.providers, store.current);
      return [...labels, '', T('menu.newProfile'), T('menu.language'), updLabel, T('menu.noLogin'), '', T('menu.exit')];
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
      colors: { [n + 1]: 'yellow', [n + 2]: 'dim', [n + 3]: 'dim', [n + 4]: 'yellow' },
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

    if (sel < 0 || sel === n + 6) return; // 退出 / Esc / q
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
    } else if (sel === n + 4) {
      // 一键免登录（铁律唯一豁免：仅写顶层 hasCompletedOnboarding=true，字节级最小修改）
      const err = markOnboardingDone();
      if (err) warnFlash = T('menu.noLoginError', err);
      else flash = T('menu.noLoginDone');
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

// profileRows 把所有配置格式化成三列菜单行：名字（主信息，默认项加粗、缺密钥变灰）
// + 备注（紧跟名字、dim，同名配置靠它区分）+ 状态（effort/缺密钥告警，dim，锚成最右一列）。
// 备注列宽按当前配置实际内容动态取最大值；状态列在最右，无需补宽。
// TUI 主菜单与 xx --list 共用此函数，保证两处呈现一致。与 Go 版 menu.go ProfileRows 对齐。
export function profileRows(providers: Provider[], current: string): string[] {
  let noteW = 0;
  for (const p of providers) {
    noteW = Math.max(noteW, displayWidth(p.note ?? ''));
  }
  if (noteW > 0) noteW += 2; // 备注列尾留出与状态列的间距
  return providers.map((p) => profileRow(p, p.name === current, noteW));
}

// rowStateText 行内状态段：只保留 effort 与「缺密钥」告警。
// 「登录态/密钥已设/密钥·API_KEY」等可用态不再用文字表达——能不能用由配置名的亮/灰区分。
function rowStateText(p: Provider): string {
  const st = getProviderState(p);
  const parts: string[] = [];
  if (st.effort) parts.push(`effort=${st.effort}`);
  if (st.key === 'noKey') parts.push(T('state.noKey'));
  return parts.join(' · ');
}

// profileRow 组成一行：名字 + 备注 + 状态三列，次要信息 dim、默认项加粗、缺密钥变灰。
// 默认项只靠名字加粗标识（不再额外画 ● 符号，与光标 ▶ 重叠冗余）；选中行由 selectMenu 整行绿。
// 亮/灰用加粗(1m)与 dim(2m)，reset 都用 22m 不动颜色，不会破坏选中行的外层绿色。
function profileRow(p: Provider, isDefault: boolean, noteW: number): string {
  const noKey = getProviderState(p).key === 'noKey';
  let name = padDisplay(providerDisplayName(p), ROW_NAME_W);
  if (noKey) name = paint(name, 'dim'); // 灰掉=当前不可用（缺密钥）
  else if (isDefault) name = paint(name, 'bold'); // 加粗=默认配置
  // 空备注=定宽空白，撑出状态列前的间距
  const note = (p.note ?? '').trim()
    ? paint(padDisplay(p.note ?? '', noteW), 'dim')
    : padDisplay('', noteW);
  // 状态在最右一列，可用且无 effort 时整列省略
  const seg = rowStateText(p);
  const state = seg ? paint(` · ${seg}`, 'dim') : '';
  return `${name}${note}${state}`;
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
