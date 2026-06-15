/**
 * 本次启用（Session-Launch）—— 进程级、阅后即焚。
 *
 * 1) 对 9 个受管键：有值 set，没值 delete（只动这 9 个）。
 * 2) 找到 claude（which；Windows 上是 claude.cmd，评审②）。
 * 3) spawn 且 stdio:inherit —— 子进程继承真实控制台句柄，天然没有现版 PowerShell 的
 *    「stdin 被包成管道 → claude 误判非交互」问题。
 * 4) Windows 不能直接 spawn .cmd（Node 的 EINVAL 防护），用 shell:true 经 cmd.exe 启动并对路径加引号。
 */
import { spawnSync } from 'node:child_process';
import which from 'which';

import { KNOWN_KEYS, getProviderEnvMap, type Provider } from '../config/store.js';

/** 把目标配置的受管环境变量套到当前进程（有值 set、没值 delete，只动这 9 个）。 */
export function applyManagedEnv(p: Provider): void {
  const map = getProviderEnvMap(p);
  for (const key of KNOWN_KEYS) {
    const v = map[key];
    if (typeof v === 'string' && v.trim() !== '') process.env[key] = v;
    else delete process.env[key];
  }
}

/** 在 PATH 中定位 claude；找不到返回 null。Windows 下通常解析到 claude.cmd。 */
export function resolveClaude(): string | null {
  return which.sync('claude', { nothrow: true });
}

export interface LaunchResult {
  /** claude 不在 PATH。 */
  claudeMissing?: boolean;
  /** spawn 自身失败（非 claude 的退出码）。 */
  spawnError?: Error;
  /** claude 的退出码（正常退出时）。 */
  status: number | null;
}

/**
 * 套环境 + 启动 claude，阻塞直到其退出。调用方负责在前后打印 banner / 处理 claudeMissing。
 * @param claudePath 可注入（测试用）；缺省走 resolveClaude()。
 */
export function sessionLaunch(p: Provider, claudePath?: string): LaunchResult {
  applyManagedEnv(p);
  const bin = claudePath ?? resolveClaude();
  if (!bin) return { claudeMissing: true, status: null };

  const isWin = process.platform === 'win32';
  // Windows：经 cmd.exe 启动以兼容 .cmd 包装，路径加引号防空格；Unix：直接 exec 解析后的真实路径。
  const file = isWin ? `"${bin}"` : bin;
  const res = spawnSync(file, [], { stdio: 'inherit', shell: isWin });
  if (res.error) return { spawnError: res.error, status: null };
  return { status: res.status };
}
