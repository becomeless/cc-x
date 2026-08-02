/**
 * 对 ~/.claude.json 的受控访问。
 * 铁律：ccx 从不写 Claude Code 配置文件。唯一豁免（用户主动触发的主菜单「一键免登录」）：
 * 把顶层布尔字段 hasCompletedOnboarding 写为 true——字符串级字节最小修改，绝不整文件 JSON 重排，
 * 文件不合法 JSON 时拒绝写入。除此外仍只读不写。
 *
 * 实现必须用 Buffer 字节操作（readFileSync 不带 encoding），绝不用字符串：字符串解码会把非法字节
 * 替换成 U+FFFD、且 JS 字符串索引是 UTF-16 码元与字节偏移不一致，重写会污染文件。与 Go 版
 * internal/claudecfg/onboarding.go 逐字节对齐。
 */
import { chmodSync, readFileSync, realpathSync, renameSync, statSync, writeFileSync } from 'node:fs';
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

/** 同目录 temp + rename 原子写；已存在时保留原权限位（Windows 上 chmod 仅只读位，可忽略）。 */
function writeAtomic(path: string, data: Buffer): void {
  const tmp = `${path}.tmp-${process.pid}`;
  writeFileSync(tmp, data);
  try {
    chmodSync(tmp, statSync(path).mode);
  } catch {
    // 原文件不存在或 chmod 失败：跳过
  }
  renameSync(tmp, path);
}
