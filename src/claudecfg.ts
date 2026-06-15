/**
 * 对 ~/.claude.json 的只读访问。
 * 铁律：只读不写，任何写操作都在 ccx 范围之外。
 */
import { existsSync, readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

/**
 * 只读检测 ~/.claude.json 的 hasCompletedOnboarding 字段。
 * 文件不存在、解析失败或字段缺失，均返回 false（视为未完成）。
 */
export function hasCompletedOnboarding(): boolean {
  const path = join(homedir(), '.claude.json');
  if (!existsSync(path)) return false;
  try {
    const data = JSON.parse(readFileSync(path, 'utf-8')) as Record<string, unknown>;
    return data.hasCompletedOnboarding === true;
  } catch {
    return false;
  }
}
