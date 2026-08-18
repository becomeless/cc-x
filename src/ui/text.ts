/**
 * 文本输入。**不用 inquirer**——它的 readline 会和自绘 selectMenu 的 raw 模式抢 stdin、收不到按键。
 * 改回现版 PS 的两套机制（同一时刻只有一套在跑，互不干扰）：
 *
 *  · readValue：raw 逐键行编辑器（与 selectMenu 同机制，已验证可用），用于 ASCII 字段（密钥/模型/effort）。
 *               密钥回显 `*`。语义对齐 Read-Value：回车空=不改、`-`=清空、Esc=取消、Ctrl+C 退出。
 *  · readText ：cooked 模式 readline（兼容中文输入法，评审④），用于中文字段（备注/自定义名/手动地址）。
 */
import { createInterface, emitKeypressEvents, type Key } from 'node:readline';
import stringWidth from 'string-width';

import { T } from '../i18n/index.js';

export interface ReadResult {
  changed: boolean;
  value: string;
}

/** 行编辑器状态：字符按**码点**存（代理对字符不会被拆成半个），pos 为光标位（0..chars.length）。 */
export interface LineEditState {
  chars: string[];
  pos: number;
}

export type LineEditOp = 'insert' | 'backspace' | 'delete' | 'left' | 'right' | 'home' | 'end';

// 组合附加符宽度为 0（不单独占列），不钳成 1：与 tailW 的整串宽度保持一致，
// 光标跨过组合符无位移、重绘按真实列数对齐。已知残留：退格/Delete 直接删 0 宽组合符时
// 屏上附标滞留到下一次编辑重绘（缓冲值始终正确）。
const dispW = (r: string, secret: boolean): number => (secret ? 1 : stringWidth(r));
const dispRune = (r: string, secret: boolean): string => (secret ? '*' : r);
const tailDisp = (rs: string[], secret: boolean): string => (secret ? '*'.repeat(rs.length) : rs.join(''));
const tailW = (rs: string[], secret: boolean): number => (secret ? rs.length : stringWidth(rs.join('')));

/**
 * 对行编辑器状态执行一次编辑操作（←/→/退格/Delete/光标处插入），返回新状态与回显。
 * 回显按字符**显示宽度**重绘光标后的内容：全角字符占 2 列，只擦 1 列会留下半个字形，
 * 用户为清掉残影再按退格就会误删前一个真实字符（保存后「少一两个字母」的直接原因之一）。
 * secret 时屏上每字符是一个 `*`（1 列）。
 */
export function applyLineEdit(
  state: LineEditState,
  op: LineEditOp,
  text = '',
  secret = false,
): { chars: string[]; pos: number; echo: string } {
  const chars = state.chars.slice();
  let pos = state.pos;
  let echo = '';
  switch (op) {
    case 'left':
      if (pos > 0) {
        pos--;
        echo = '\b'.repeat(dispW(chars[pos]!, secret));
      }
      break;
    case 'right':
      if (pos < chars.length) {
        echo = dispRune(chars[pos]!, secret);
        pos++;
      }
      break;
    case 'home':
      if (pos > 0) {
        echo = '\b'.repeat(tailW(chars.slice(0, pos), secret)); // 退到行首
        pos = 0;
      }
      break;
    case 'end':
      if (pos < chars.length) {
        echo = tailDisp(chars.slice(pos), secret); // 重打光标后的内容，光标随之前进到行尾
        pos = chars.length;
      }
      break;
    case 'backspace': {
      if (pos === 0) break;
      const removed = chars[pos - 1]!;
      chars.splice(pos - 1, 1);
      pos--;
      const w = dispW(removed, secret);
      const tail = chars.slice(pos);
      echo = '\b'.repeat(w) + tailDisp(tail, secret) + ' '.repeat(w) + '\b'.repeat(tailW(tail, secret) + w);
      break;
    }
    case 'delete': {
      if (pos === chars.length) break;
      const removed = chars[pos]!;
      chars.splice(pos, 1);
      const w = dispW(removed, secret);
      const tail = chars.slice(pos);
      echo = tailDisp(tail, secret) + ' '.repeat(w) + '\b'.repeat(tailW(tail, secret) + w);
      break;
    }
    case 'insert': {
      const parts = [...text];
      if (parts.length === 0) break;
      chars.splice(pos, 0, ...parts);
      pos += parts.length;
      const tail = chars.slice(pos);
      echo = (secret ? '*'.repeat(parts.length) : text) + tailDisp(tail, secret) + '\b'.repeat(tailW(tail, secret));
      break;
    }
  }
  return { chars, pos, echo };
}

/** raw 逐键读 ASCII 字段：回车空=不改、`-`=清空、其它=替换、Esc/Ctrl+C=取消。
 * 支持光标编辑：←/→ 移动光标、光标处插入、退格删光标前字符、Delete 删光标处字符。 */
export async function readValue(label: string, current: string, secret = false): Promise<ReadResult> {
  const stdin = process.stdin;
  const stdout = process.stdout;
  const cur = current === '' ? T('empty.paren') : secret ? '********' : current;
  stdout.write(`\n  ${label}  [${T('edit.current', cur)}]  ${T('edit.inputHint')}\n  > `);

  if (!stdin.isTTY) {
    const line = await readText(''); // 非交互回退
    if (line === undefined || line === '') return { changed: false, value: current };
    if (line === '-') return { changed: true, value: '' };
    return { changed: true, value: line };
  }

  emitKeypressEvents(stdin);
  const wasRaw = stdin.isRaw ?? false;
  stdin.setRawMode(true);
  stdin.resume();

  let state: LineEditState = { chars: [], pos: 0 };
  return new Promise<ReadResult>((resolve) => {
    const cleanup = (): void => {
      stdin.off('keypress', onKey);
      if (!wasRaw) stdin.setRawMode(false);
      stdin.pause();
      stdout.write('\n');
    };
    const apply = (op: LineEditOp, text = ''): void => {
      const r = applyLineEdit(state, op, text, secret);
      state = { chars: r.chars, pos: r.pos };
      if (r.echo) stdout.write(r.echo);
    };
    const onKey = (str: string | undefined, key: Key): void => {
      if (key?.ctrl && key.name === 'c') {
        cleanup();
        resolve({ changed: false, value: current });
        process.exit(130);
      }
      if (key?.name === 'return' || key?.name === 'enter') {
        cleanup();
        const buf = state.chars.join('');
        if (buf === '') resolve({ changed: false, value: current });
        else if (buf === '-') resolve({ changed: true, value: '' });
        else resolve({ changed: true, value: buf });
        return;
      }
      if (key?.name === 'escape') {
        cleanup();
        resolve({ changed: false, value: current });
        return;
      }
      if (key?.name === 'backspace') {
        apply('backspace');
        return;
      }
      if (key?.name === 'delete') {
        apply('delete');
        return;
      }
      if (key?.name === 'left') {
        apply('left');
        return;
      }
      if (key?.name === 'right') {
        apply('right');
        return;
      }
      if (key?.name === 'home') {
        apply('home');
        return;
      }
      if (key?.name === 'end') {
        apply('end');
        return;
      }
      // 普通字符（含粘贴的多字符），过滤控制字符
      if (str && !key?.ctrl && !key?.meta) {
        const printable = [...str].filter((c) => c >= ' ' && c !== '\x7f').join('');
        if (printable) apply('insert', printable);
      }
    };
    stdin.on('keypress', onKey);
  });
}

/** cooked 模式读一行（兼容中文输入法）；Ctrl+C/中止返回 undefined。 */
export async function readText(message: string): Promise<string | undefined> {
  const stdin = process.stdin;
  if (stdin.isTTY) stdin.setRawMode(false); // 确保 cooked，输入法才能组词
  stdin.resume();
  const rl = createInterface({ input: stdin, output: process.stdout });
  return new Promise<string | undefined>((resolve) => {
    let done = false;
    rl.on('SIGINT', () => {
      if (done) return;
      done = true;
      rl.close();
      resolve(undefined);
    });
    rl.question(message ? `${message}` : '  > ', (ans) => {
      if (done) return;
      done = true;
      rl.close();
      resolve(ans);
    });
  });
}
