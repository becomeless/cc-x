/**
 * 「检查新版本」的轻量实现，对齐 Go 版 internal/update。
 *
 * 不走 GitHub API（仅用 releases/latest 的 302 重定向抠版本号，无速率限制），结果缓存在
 * ~/.cc-mini/update-check.json，每 24h 才真去网络。显示永远读缓存（瞬时、不阻塞），过期时后台
 * 异步刷新——新版本「下次打开」才提示。离线/失败一律静默。只写工具自己的 ~/.cc-mini/（铁律）。
 */
import { existsSync, mkdirSync, readFileSync, renameSync, unlinkSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

export const MODE_OFF = '';
export const MODE_NOTIFY = 'notify';

const LATEST_URL = 'https://github.com/becomeless/cc-x/releases/latest';
const CACHE_MAX_AGE_MS = 24 * 60 * 60 * 1000;
const HTTP_TIMEOUT_MS = 2000;
const CACHE_FILE = 'update-check.json';
const UPGRADE_CMD = 'npm i -g @cc-x/cc-x@latest';

interface Cache {
  checkedAt: number; // unix 秒
  latest: string;
}

const TAG_RE = /\/tag\/v?(\d+\.\d+\.\d+)/;

/** 读缓存：若已知有比 current 更新的版本，返回最新版本号；否则 undefined。不联网。 */
export function banner(storeDir: string, current: string): string | undefined {
  const c = readCache(storeDir);
  if (!c || !c.latest) return undefined;
  return isNewer(c.latest, current) ? c.latest : undefined;
}

/** 缓存过期（或不存在）时后台异步联网刷新一次；不阻塞调用方。 */
export function maybeRefresh(storeDir: string): void {
  const c = readCache(storeDir);
  if (c && Date.now() - c.checkedAt * 1000 < CACHE_MAX_AGE_MS) return; // 仍新鲜
  void refresh(storeDir); // fire-and-forget
}

async function refresh(storeDir: string): Promise<void> {
  const latest = await fetchLatest();
  if (!latest) return; // 失败静默；不动缓存
  writeCache(storeDir, { checkedAt: Math.floor(Date.now() / 1000), latest });
}

/** 当前版本（npm 版）的升级命令。 */
export function upgradeCommand(): string {
  return UPGRADE_CMD;
}

function cachePath(storeDir: string): string {
  return join(storeDir, CACHE_FILE);
}

function readCache(storeDir: string): Cache | undefined {
  try {
    const c = JSON.parse(readFileSync(cachePath(storeDir), 'utf-8')) as Cache;
    if (typeof c.checkedAt === 'number' && typeof c.latest === 'string') return c;
  } catch {
    /* 无缓存 / 损坏 -> 当作没有 */
  }
  return undefined;
}

/** 原子写（temp + rename），避免被进程退出打断写出半截文件。
 * 写或 rename 失败都清理 tmp（对齐 Go 版 writeCache，不留孤儿文件）。 */
function writeCache(storeDir: string, c: Cache): void {
  const tmp = `${cachePath(storeDir)}.tmp`;
  try {
    if (!existsSync(storeDir)) mkdirSync(storeDir, { recursive: true });
    writeFileSync(tmp, JSON.stringify(c), 'utf-8');
    renameSync(tmp, cachePath(storeDir));
  } catch {
    try {
      unlinkSync(tmp);
    } catch {
      /* 清理失败忽略；写不了就算了 */
    }
  }
}

/** 用 releases/latest 的 302 重定向抠最新版本号；redirect:'manual' = 不跟随 = 不走 GitHub API。 */
async function fetchLatest(): Promise<string | undefined> {
  try {
    const resp = await fetch(LATEST_URL, {
      redirect: 'manual',
      signal: AbortSignal.timeout(HTTP_TIMEOUT_MS),
    });
    const loc = resp.headers.get('location');
    if (!loc) return undefined;
    const m = TAG_RE.exec(loc);
    return m ? m[1] : undefined;
  } catch {
    return undefined;
  }
}

/** latest 是否严格新于 current（"a.b.c"，忽略前导 v 与后缀）。解析失败一律 false（不误报）。 */
export function isNewer(latest: string, current: string): boolean {
  const lp = parseSemver(latest);
  const cp = parseSemver(current);
  if (!lp || !cp) return false;
  const [la, lb, lc] = lp;
  const [ca, cb, cc] = cp;
  if (la !== ca) return la > ca;
  if (lb !== cb) return lb > cb;
  return lc > cc;
}

function parseSemver(s: string): [number, number, number] | undefined {
  let v = s.trim().replace(/^v/, '');
  const cut = v.search(/[-+]/);
  if (cut >= 0) v = v.slice(0, cut);
  const parts = v.split('.');
  if (parts.length !== 3) return undefined;
  const nums = parts.map((p) => Number.parseInt(p, 10));
  if (nums.some((n) => Number.isNaN(n))) return undefined;
  return [nums[0]!, nums[1]!, nums[2]!];
}
