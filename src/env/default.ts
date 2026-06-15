/**
 * 设为默认（Set-Default）—— 持久化用户环境变量，仅影响新开终端、不动运行中会话。
 *
 * 对 9 个受管键：目标配置有值的写值，没值的清除（vals 里记 null）。然后 store.current=name 并存盘。
 * 平台分叉是唯一有平台差异的地方：Windows 走注册表+广播，Unix 走 rc 文件 marker 块。
 * `--default-scope process` = 不落盘 dry-run（评审⑥）：照常算 vals、更新 store，但跳过系统持久化。
 */
import { KNOWN_KEYS, getProviderEnvMap, saveStore, type Provider, type Store, type StorePaths } from '../config/store.js';
import { persistUnix, type UnixPersistResult } from './persist-unix.js';
import { persistWindows, type EnvVals, type PersistResult } from './persist-windows.js';

export type DefaultScope = 'user' | 'process';

export interface SetDefaultResult {
  scope: DefaultScope;
  /** scope==='process'：未改系统，仅更新存储。 */
  dryRun: boolean;
  windows?: PersistResult;
  unix?: UnixPersistResult;
}

/** 每个受管键 → 值或 null（清除）。对齐现版 Set-Default 里的 $vals 构造。 */
export function computeManagedVals(p: Provider): EnvVals {
  const map = getProviderEnvMap(p);
  const vals: EnvVals = {};
  for (const k of KNOWN_KEYS) {
    const v = map[k];
    vals[k] = typeof v === 'string' && v.trim() !== '' ? v : null;
  }
  return vals;
}

export function persistDefaultEnv(p: Provider, scope: DefaultScope): SetDefaultResult {
  const vals = computeManagedVals(p);
  const dryRun = scope === 'process';
  const result: SetDefaultResult = { scope, dryRun };
  if (!dryRun) {
    if (process.platform === 'win32') {
      result.windows = persistWindows(vals);
    } else {
      result.unix = persistUnix(vals);
    }
  }
  return result;
}

function envPersisted(result: SetDefaultResult): boolean {
  if (result.dryRun) return true;
  if (result.windows) return result.windows.ok;
  if (result.unix) return !result.unix.unsupported; // fish 未写入 → 不算持久化成功
  return false;
}

export function setDefault(paths: StorePaths, store: Store, p: Provider, scope: DefaultScope): SetDefaultResult {
  const result = persistDefaultEnv(p, scope);
  // [P1] 仅当持久化成功（或 dry-run）才改默认指向并落盘，避免「报失败却已改默认」的不一致。
  if (envPersisted(result)) {
    store.current = p.name;
    saveStore(paths, store);
  }
  return result;
}
