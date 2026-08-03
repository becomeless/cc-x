/**
 * 对 ~/.claude.json 的受控访问。
 * 配置边界：API、密钥、模型映射、MCP、插件、Hooks 等配置绝不由 ccx 写入。
 * 唯一允许的写入是用户主动触发主菜单「一键免登录」时，把顶层布尔字段
 * hasCompletedOnboarding 写为 true——字符串级字节最小原子修改，绝不整文件 JSON 重排，
 * 文件不合法 JSON 时拒绝写入。不得扩展此例外。
 *
 * 实现必须用 Buffer 字节操作（readFileSync 不带 encoding），绝不用字符串：字符串解码会把非法字节
 * 替换成 U+FFFD、且 JS 字符串索引是 UTF-16 码元与字节偏移不一致，重写会污染文件。与 Go 版
 * internal/claudecfg/onboarding.go 逐字节对齐。
 */
import { randomBytes } from 'node:crypto';
import { chmodSync, readFileSync, realpathSync, renameSync, statSync, unlinkSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

const KEY = Buffer.from('hasCompletedOnboarding');
const TRUE = Buffer.from('true');
const NEW_FILE = Buffer.from('{"hasCompletedOnboarding": true}\n');

function isSpace(c: number): boolean {
  return c === 0x20 || c === 0x09 || c === 0x0a || c === 0x0d;
}

function isAlpha(c: number): boolean {
  return (c >= 0x61 && c <= 0x7a) || (c >= 0x41 && c <= 0x5a);
}

function isNumChar(c: number): boolean {
  return (c >= 0x30 && c <= 0x39) || c === 0x2d || c === 0x2b || c === 0x2e || c === 0x65 || c === 0x45;
}

/** 定位 buf[v] 起的一个 JSON 值的结束位置（不含）。v 指向值首字节。返回 [start, end] 或 null。 */
function valueSpan(buf: Buffer, v: number): [number, number] | null {
  if (v >= buf.length) return null;
  const c = buf[v]!; // v < buf.length 已由调用方保证
  if (c === 0x22) {
    // 字符串：转义感知到闭合引号
    let es = false;
    for (let e = v + 1; e < buf.length; e++) {
      const b = buf[e]!; // e < buf.length 由循环条件保证
      if (es) es = false;
      else if (b === 0x5c) es = true;
      else if (b === 0x22) return [v, e + 1];
    }
    return null;
  }
  if (c === 0x7b || c === 0x5b) {
    // 对象/数组：深度匹配
    let d = 1;
    let inStr = false;
    let es = false;
    for (let e = v + 1; e < buf.length; e++) {
      const b = buf[e]!; // e < buf.length 由循环条件保证
      if (inStr) {
        if (es) es = false;
        else if (b === 0x5c) es = true;
        else if (b === 0x22) inStr = false;
        continue;
      }
      if (b === 0x22) inStr = true;
      else if (b === 0x7b || b === 0x5b) d++;
      else if (b === 0x7d || b === 0x5d) {
        d--;
        if (d === 0) return [v, e + 1];
      }
    }
    return null;
  }
  if (c === 0x74 || c === 0x66 || c === 0x6e) {
    // true/false/null：字母串
    let e = v;
    while (e < buf.length && isAlpha(buf[e]!)) e++;
    return [v, e];
  }
  if (c === 0x2d || (c >= 0x30 && c <= 0x39)) {
    // 数字
    let e = v;
    while (e < buf.length && isNumChar(buf[e]!)) e++;
    return [v, e];
  }
  return null;
}

/**
 * spliceOnboarding 单遍扫描：把顶层 hasCompletedOnboarding 的值替换为 true。
 * 顶层无该键时插入到第一个 { 之后（空对象 {} 不带尾逗号）。重复顶层键按 JSON 语义取最后一次。
 * 调用方须保证 buf 是合法 JSON 且顶层为对象。返回 [新字节, changed] 或错误消息。
 */
function spliceOnboarding(buf: Buffer): [Buffer, boolean] | string {
  let depth = 0;
  let inStr = false;
  let escaped = false;
  let lastStart = -1;
  let lastEnd = -1;
  let found = false;
  let braceAt = -1;

  for (let i = 0; i < buf.length; i++) {
    const c = buf[i]!; // i < buf.length 由循环条件保证
    if (inStr) {
      if (escaped) escaped = false;
      else if (c === 0x5c) escaped = true;
      else if (c === 0x22) inStr = false;
      continue;
    }
    if (c === 0x22) {
      if (depth === 1) {
        // 候选键：解析到闭合引号，键后跳过空白必须紧跟 : 才算键
        let end = i + 1;
        let es = false;
        while (end < buf.length) {
          const b = buf[end]!; // end < buf.length 由循环条件保证
          if (es) es = false;
          else if (b === 0x5c) es = true;
          else if (b === 0x22) break;
          end++;
        }
        if (end >= buf.length) return 'JSON 结构异常：未闭合的字符串';
        let colon = end + 1;
        while (colon < buf.length && isSpace(buf[colon]!)) colon++;
        if (colon < buf.length && buf[colon]! === 0x3a && buf.subarray(i + 1, end).equals(KEY)) {
          let v = colon + 1;
          while (v < buf.length && isSpace(buf[v]!)) v++;
          const span = valueSpan(buf, v);
          if (!span) return 'JSON 结构异常：无法定位值';
          [lastStart, lastEnd] = span;
          found = true;
          i = lastEnd - 1;
          continue;
        }
        i = end;
        continue;
      }
      inStr = true;
    } else if (c === 0x7b) {
      if (braceAt === -1) braceAt = i;
      depth++;
    } else if (c === 0x7d) {
      depth--;
    } else if (c === 0x5b) {
      depth++;
    } else if (c === 0x5d) {
      depth--;
    }
  }

  if (found) {
    if (buf.subarray(lastStart, lastEnd).equals(TRUE)) return [buf, false];
    return [Buffer.concat([buf.subarray(0, lastStart), TRUE, buf.subarray(lastEnd)]), true];
  }
  // 未命中：插入到顶层 { 之后
  let next = braceAt + 1;
  while (next < buf.length && isSpace(buf[next]!)) next++;
  const ins = next < buf.length && buf[next]! === 0x7d
    ? Buffer.from('"hasCompletedOnboarding": true') // 空对象：不加尾逗号
    : Buffer.from('"hasCompletedOnboarding": true,');
  return [Buffer.concat([buf.subarray(0, braceAt + 1), ins, buf.subarray(braceAt + 1)]), true];
}

/** 一键免登录：把 ~/.claude.json 顶层 hasCompletedOnboarding 写为 true。返回错误消息；undefined = 成功。 */
export function markOnboardingDone(): string | undefined {
  return markOnboardingDoneIn(join(homedir(), '.claude.json'));
}

/** 显式路径实现（测试用）。文件不存在则创建仅含该字段的最小文件；任何失败都保持文件原样。 */
export function markOnboardingDoneIn(path: string): string | undefined {
  // 符号链接先解析再写（temp+rename 直接写会替换链接本体）；文件不存在则保持原路径（走创建）。
  try {
    path = realpathSync(path);
  } catch {
    // 保持原路径
  }
  let buf: Buffer;
  try {
    buf = readFileSync(path);
  } catch (e) {
    const code = (e as NodeJS.ErrnoException).code;
    if (code === 'ENOENT') {
      try {
        writeAtomic(path, NEW_FILE);
        return undefined;
      } catch (e2) {
        return String((e2 as Error).message);
      }
    }
    return String((e as Error).message);
  }
  // 校验合法性与顶层形状（仅校验，不解析重排）
  try {
    JSON.parse(buf.toString('utf8'));
  } catch {
    return `${path} 不是合法 JSON，拒绝改动`;
  }
  if (buf[firstNonSpaceAt(buf)]! !== 0x7b) {
    return `${path} 顶层不是 JSON 对象，拒绝改动`;
  }
  const spliced = spliceOnboarding(buf);
  if (typeof spliced === 'string') return spliced;
  const [next, changed] = spliced;
  if (!changed) return undefined; // 值已是 true：幂等，不写盘
  try {
    writeAtomic(path, next);
    return undefined;
  } catch (e) {
    return String((e as Error).message);
  }
}

function firstNonSpaceAt(buf: Buffer): number {
  let i = 0;
  while (i < buf.length && isSpace(buf[i]!)) i++;
  return i;
}

/** 同目录 temp + rename 原子写；新建文件固定 0600（敏感配置文件，不可被其他本机用户读取），
 * 已存在时保留原权限位（Windows 上 chmod 仅只读位，可忽略）。与 Go 版 atomicWrite
 * （CreateTemp 默认 0600 + O_EXCL 独占创建 + 已存在时 Chmod 原权限 + defer 清理）逐行对齐。
 * 临时名随机（非 PID 可预测）+ flag:'wx' 独占创建：不会复用 PID 重用/崩溃残留的同名旧文件，
 * 也不会跟随预先存在的符号链接。任何失败（write/chmod/rename）都清理 temp，不留孤儿文件；
 * 清理先 chmod 恢复可写再 unlink，且两步骤独立执行——chmod 失败不阻挡 unlink
 * （失败路径上 temp 可能已被 chmod 成原文件只读位，Windows 上直接 unlink 只读文件会失败残留）。
 * 注：POSIX 上原子替换可以覆盖只读目标（rename 取决于父目录权限），但会保留其权限位；
 * Windows 上替换是否成功取决于目标文件属性与占用状态。 */
function writeAtomic(path: string, data: Buffer): void {
  const tmp = `${path}.tmp-${randomBytes(6).toString('hex')}`;
  try {
    writeFileSync(tmp, data, { mode: 0o600, flag: 'wx' });
    let originalMode: number | undefined;
    try {
      originalMode = statSync(path).mode & 0o777;
    } catch (e) {
      if (!isErrnoException(e, 'ENOENT')) throw e;
    }
    if (originalMode !== undefined) chmodSync(tmp, originalMode);
    renameSync(tmp, path);
  } catch (e) {
    try {
      chmodSync(tmp, 0o600);
    } catch {
      // chmod 失败也继续尝试 unlink（恢复可写只是手段，删除仍是目标）
    }
    try {
      unlinkSync(tmp);
    } catch {
      // 清理失败忽略（原异常优先）
    }
    throw e;
  }
}

function isErrnoException(e: unknown, code: string): boolean {
  return typeof e === 'object' && e !== null && 'code' in e && (e as NodeJS.ErrnoException).code === code;
}
