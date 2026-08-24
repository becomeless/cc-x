/**
 * 自绘 ↑↓ 选择菜单（对齐现版 Select-Menu）。
 *
 * - ↑↓ 导航（跳过 '' 分隔空行）、数字键直选、Enter 确认、q/Esc 取消、Ctrl+C 退出。
 * - 原地重绘（光标上移 + 清屏到底 + 隐藏光标）→ 不闪烁。
 * - 可选就地排序：传 onMove + movableCount，Shift+↑↓ 或 PgUp/PgDn 在顶部前 N 项内移动选中项。
 * - 非交互/无 TTY 时回退到「打印列表 + 读一行序号」。
 *
 * 注：文本输入（含中文）不在这里——那走 ui/text.ts（raw readValue / cooked readText，后者兼容输入法，评审④）。
 */
import { createInterface, emitKeypressEvents, type Key } from 'node:readline';

import { T } from '../i18n/index.js';
import { CLEAR_DOWN, CLEAR_SCREEN, CR, cursorUp, HIDE_CURSOR, paint, SHOW_CURSOR, type Color } from '../utils/ansi.js';
import { truncateDisplay } from '../utils/display.js';

export interface SelectOptions {
  title?: string;
  /** 标题下方黄字横幅（如「有新版本」），常驻显示。 */
  notice?: string;
  /** 菜单项；'' = 不可选的分隔空行（导航跳过）。 */
  items: string[];
  hint?: string;
  /** 顶部绿色 toast（如「已设为默认」），显示一轮。 */
  status?: string;
  /** 初始选中项（记忆选中）。 */
  start?: number;
  /** 按索引上色（如「＋ 新增配置」用黄色）。 */
  colors?: Record<number, Color>;
  /** 顶部可排序区的项数（仅前 N 项可被 Shift+↑↓/PgUp/PgDn 移动）。 */
  movableCount?: number;
  /** 排序回调：交换数据并返回重建后的菜单项标签数组。 */
  onMove?: (from: number, to: number) => string[];
  /** 单键快捷键：返回新索引并确认，-1=忽略。 */
  onKey?: (r: string, idx: number) => number;
  /** 关闭行首序号（默认显示，与数字直选一致；编辑表单项多于 9 个时关闭）。 */
  noNumber?: boolean;
}

/** 返回选中索引；取消（q/Esc/非法）返回 -1。 */
export async function selectMenu(opts: SelectOptions): Promise<number> {
  const stdin = process.stdin;
  const stdout = process.stdout;
  let items = opts.items.slice();

  if (!stdin.isTTY || !stdout.isTTY) return fallbackSelect(opts, items);

  const nextSel = (i: number, d: number): number => {
    const n = items.length;
    do {
      i = (i + d + n) % n;
    } while (items[i] === '');
    return i;
  };
  let idx = opts.start ?? 0;
  if (idx < 0 || idx >= items.length || items[idx] === '') idx = nextSel(Math.max(0, Math.min(idx, items.length - 1)), 1);

  emitKeypressEvents(stdin);
  const wasRaw = stdin.isRaw ?? false;
  stdin.setRawMode(true);
  stdin.resume();
  stdout.write(CLEAR_SCREEN + HIDE_CURSOR); // 清屏归位，制造「整页」感（每进一级菜单都是新页）

  let prevLines = 0;
  let top = 0; // 滚动窗口首项下标：保证选中项 idx 始终落在窗口内，随上下移动滑窗
  const render = (): void => {
    const cols = stdout.columns ?? 80;
    const rows = stdout.rows ?? 24;
    const lines: string[] = [''];
    if (opts.title) {
      lines.push(`  ${paint(opts.title, 'cyan')}`, '');
    }
    if (opts.notice) {
      for (const line of splitNonEmptyLines(opts.notice)) lines.push(`  ${paint(line, 'yellow')}`);
      lines.push('');
    }
    if (opts.status) {
      for (const line of splitNonEmptyLines(opts.status)) lines.push(`  ${paint(line, 'green')}`);
      lines.push('');
    }
    const headerN = lines.length;

    // 项窗口行数 = 终端可视高度 - 头部 - 尾部；预留 1 行，避免光标落在底行时回滚抖动。
    const footerBase = opts.hint ? 2 : 1;
    // 先按「未加位置指示」估视口，判断 item 数是否溢出（会话内 item 数固定，此判断稳定）。
    const baseViewH = Math.max(1, rows - headerN - footerBase - 1);
    const scrollable = items.length > baseViewH;
    let footerN = footerBase;
    let indicator = '';
    if (scrollable) {
      indicator = `${idx + 1} / ${items.length}`;
      footerN += 1;
    }
    const viewH = Math.max(1, rows - headerN - footerN - 1);
    const { start, end } = scrollWindow(top, idx, viewH, items.length);
    top = start;

    // 项窗口：只渲染 [start,end)，选中项随滑窗保证可见。
    //（原逻辑把整棵 items 全画出来，超过屏高后选中项滚出视口、屏上「不动」。）
    for (let i = start; i < end; i++) {
      const it = items[i] ?? '';
      if (it === '') {
        lines.push('');
        continue;
      }
      const num = opts.noNumber ? '' : `${i + 1}. `; // 行首序号，与数字键直选对应
      if (i === idx) lines.push(paint(`   ▶ ${num}${it}`, 'green'));
      else lines.push(paint(`     ${num}${it}`, opts.colors?.[i] ?? 'none'));
    }
    lines.push('');
    if (indicator) lines.push(`  ${paint(indicator, 'dim')}`);
    if (opts.hint) lines.push(`  ${paint(opts.hint, 'dim')}`);

    const block = lines.map((l) => truncateDisplay(l, cols - 1)).join('\n');
    const prefix = prevLines > 0 ? cursorUp(prevLines - 1) + CR + CLEAR_DOWN : '';
    stdout.write(prefix + block);
    prevLines = lines.length;
  };

  return new Promise<number>((resolve) => {
    const cleanup = (result: number): void => {
      stdin.off('keypress', onKey);
      stdout.write(`${SHOW_CURSOR}\n`);
      if (!wasRaw) stdin.setRawMode(false);
      stdin.pause();
      resolve(result);
    };
    const onKey = (str: string | undefined, key: Key): void => {
      if (key?.ctrl && key.name === 'c') {
        cleanup(-1);
        process.exit(130);
      }
      // 就地排序（仅可排序区内）
      if (opts.onMove && opts.movableCount) {
        const up = key?.name === 'pageup' || (Boolean(key?.shift) && key?.name === 'up');
        const down = key?.name === 'pagedown' || (Boolean(key?.shift) && key?.name === 'down');
        if (up && idx > 0 && idx < opts.movableCount) {
          items = opts.onMove(idx, idx - 1);
          idx -= 1;
          render();
          return;
        }
        if (down && idx < opts.movableCount - 1) {
          items = opts.onMove(idx, idx + 1);
          idx += 1;
          render();
          return;
        }
      }
      switch (key?.name) {
        case 'up':
          idx = nextSel(idx, -1);
          render();
          return;
        case 'down':
          idx = nextSel(idx, 1);
          render();
          return;
        case 'return':
        case 'enter':
          cleanup(idx);
          return;
        case 'escape':
          cleanup(-1);
          return;
        default:
          break;
      }
      const ch = str ?? '';
      if (!opts.noNumber && /^[0-9]$/.test(ch)) {
        const n = Number.parseInt(ch, 10);
        if (n >= 1 && n <= items.length && items[n - 1] !== '') cleanup(n - 1);
        return;
      }
      if (ch === 'q') {
        cleanup(-1);
        return;
      }
      if (opts.onKey) {
        const r = opts.onKey(ch, idx);
        if (r >= 0) cleanup(r);
      }
    };
    stdin.on('keypress', onKey);
    render();
  });
}

/** 非交互回退：打印列表 + 读一行序号。 */
async function fallbackSelect(opts: SelectOptions, items: string[]): Promise<number> {
  const stdout = process.stdout;
  stdout.write('\n');
  if (opts.title) stdout.write(`  ${opts.title}\n\n`);
  if (opts.notice) {
    splitNonEmptyLines(opts.notice).forEach((line) => stdout.write(`  ${line}\n`));
    stdout.write('\n');
  }
  if (opts.status) {
    splitNonEmptyLines(opts.status).forEach((line) => stdout.write(`  ${line}\n`));
    stdout.write('\n');
  }
  const indexMap: number[] = [];
  items.forEach((it, i) => {
    if (it !== '') {
      indexMap.push(i);
      stdout.write(`   ${indexMap.length}. ${it}\n`);
    }
  });
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  const ans = await new Promise<string>((res) => rl.question(`  ${T('menu.prompt')}`, res));
  rl.close();
  const t = ans.trim();
  if (t === 'q') return -1;
  if (/^\d+$/.test(t)) {
    const n = Number.parseInt(t, 10);
    if (n >= 1 && n <= indexMap.length) return indexMap[n - 1] ?? -1;
  }
  return -1;
}

function splitNonEmptyLines(s: string): string[] {
  return s
    .split('\n')
    .map((line) => line.trimEnd())
    .filter((line) => line.trim() !== '');
}

/** 滚动窗口 [start,end)：高 viewH 行，始终包含选中项 idx。与 Go 版 scrollWindow 对齐。 */
export function scrollWindow(top: number, idx: number, viewH: number, n: number): { start: number; end: number } {
  if (viewH < 1) viewH = 1;
  if (idx < top) top = idx;
  else if (idx >= top + viewH) top = idx - viewH + 1;
  if (top < 0) top = 0;
  if (top > n - viewH) top = n - viewH;
  if (top < 0) top = 0;
  const end = Math.min(top + viewH, n);
  return { start: top, end };
}

/**
 * raw 模式下读一个按键确认（y/Y=是，其余任意键=否），无需回车，与菜单 raw 体验一致。
 * 非 TTY 时回退到 cooked 读行。对应 Go 版 confirmKey。
 */
export async function confirmKey(prompt: string): Promise<boolean> {
  const stdin = process.stdin;
  const stdout = process.stdout;
  if (!stdin.isTTY || !stdout.isTTY) {
    const rl = createInterface({ input: stdin, output: stdout });
    const ans = await new Promise<string>((res) => rl.question(`  ${prompt}`, res));
    rl.close();
    const t = ans.trim();
    return t === 'y' || t === 'Y';
  }
  stdout.write(`  ${prompt}`);
  emitKeypressEvents(stdin);
  const wasRaw = stdin.isRaw ?? false;
  stdin.setRawMode(true);
  stdin.resume();
  return new Promise<boolean>((resolve) => {
    const onKey = (str: string | undefined, key: Key): void => {
      stdin.off('keypress', onKey);
      if (!wasRaw) stdin.setRawMode(false);
      stdin.pause();
      stdout.write('\n');
      if (key?.ctrl && key.name === 'c') {
        resolve(false);
        return;
      }
      resolve((str ?? '').toLowerCase() === 'y');
    };
    stdin.on('keypress', onKey);
  });
}
