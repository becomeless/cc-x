/**
 * 共享动作：本次启用（CLI 路径与菜单都用，避免 index ↔ menus 循环依赖）。
 */
import { getProviderState, type Provider } from './config/store.js';
import { sessionLaunch } from './env/session.js';
import { providerDisplayName, T } from './i18n/index.js';

/** 非官方且未填密钥时给黄字提示（对齐现版）。 */
export function warnIfNoKey(p: Provider): void {
  if (getProviderState(p).key === 'noKey') {
    console.error(`  ${T('session.noKey', providerDisplayName(p))}`);
  }
}

/** 本次启用：缺密钥提示 + banner + 套环境启动 claude，阻塞至其退出。 */
export function launchSession(p: Provider): void {
  warnIfNoKey(p);
  console.log('');
  console.log(`  ${T('session.launch', providerDisplayName(p))}`);
  console.log(`  ${T('session.starting')}`);
  console.log('');
  const res = sessionLaunch(p);
  if (res.claudeMissing) {
    console.error(`  ${T('session.noClaude')}`);
    console.error(`  ${T('session.noClaudeHint')}`);
    process.exitCode = 1;
    return;
  }
  if (res.spawnError) {
    console.error(`  ${res.spawnError.message}`);
    process.exitCode = 1;
    return;
  }
  if (typeof res.status === 'number' && res.status !== 0) process.exitCode = res.status;
}
