import assert from 'node:assert/strict';
import test from 'node:test';

import { applyLineEdit, type LineEditState } from './text.js';

const st = (s: string, pos: number): LineEditState => ({ chars: [...s], pos });

// 回归：全角字符（如用户输入的「【」）占 2 列；退格只擦 1 列会留下半个字形，
// 用户为清掉残影再按退格就会误删前一个真实字符——即「保存后少一两个字母」的直接原因之一。
test('applyLineEdit: 光标处插入 AB→ACB', () => {
  const r = applyLineEdit(st('AB', 1), 'insert', 'C');
  assert.equal(r.chars.join(''), 'ACB');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, 'CB\b');
});

test('applyLineEdit: 末尾插入等价追加', () => {
  const r = applyLineEdit(st('AB', 2), 'insert', 'C');
  assert.equal(r.chars.join(''), 'ABC');
  assert.equal(r.pos, 3);
  assert.equal(r.echo, 'C');
});

test('applyLineEdit: 中插全角字符按 2 列重绘', () => {
  const r = applyLineEdit(st('AB', 1), 'insert', '【');
  assert.equal(r.chars.join(''), 'A【B');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, '【B\b');
});

test('applyLineEdit: 粘贴多字符一次插入', () => {
  const r = applyLineEdit(st('', 0), 'insert', 'abc');
  assert.equal(r.chars.join(''), 'abc');
  assert.equal(r.pos, 3);
  assert.equal(r.echo, 'abc');
});

test('applyLineEdit: 尾部退格全角字符擦 2 列', () => {
  const r = applyLineEdit(st('mimo【', 5), 'backspace');
  assert.equal(r.chars.join(''), 'mimo');
  assert.equal(r.pos, 4);
  assert.equal(r.echo, '\b\b  \b\b');
});

test('applyLineEdit: 中间退格 ACB→AB', () => {
  const r = applyLineEdit(st('ACB', 2), 'backspace');
  assert.equal(r.chars.join(''), 'AB');
  assert.equal(r.pos, 1);
  assert.equal(r.echo, '\bB \b\b');
});

test('applyLineEdit: 光标在行首退格无效', () => {
  const r = applyLineEdit(st('AB', 0), 'backspace');
  assert.equal(r.chars.join(''), 'AB');
  assert.equal(r.pos, 0);
  assert.equal(r.echo, '');
});

test('applyLineEdit: Delete 删光标处字符 ACB→AB', () => {
  const r = applyLineEdit(st('ACB', 1), 'delete');
  assert.equal(r.chars.join(''), 'AB');
  assert.equal(r.pos, 1);
  assert.equal(r.echo, 'B \b\b');
});

test('applyLineEdit: 光标在行尾 Delete 无效', () => {
  const r = applyLineEdit(st('AB', 2), 'delete');
  assert.equal(r.chars.join(''), 'AB');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, '');
});

test('applyLineEdit: ←/→ 按字符宽度移动', () => {
  const left = applyLineEdit(st('A【', 2), 'left');
  assert.equal(left.pos, 1);
  assert.equal(left.echo, '\b\b'); // 跨过 2 列的全角字符
  const right = applyLineEdit({ chars: left.chars, pos: left.pos }, 'right');
  assert.equal(right.pos, 2);
  assert.equal(right.echo, '【');
});

test('applyLineEdit: 行首 ← / 行尾 → 无效', () => {
  assert.equal(applyLineEdit(st('A', 0), 'left').echo, '');
  assert.equal(applyLineEdit(st('A', 1), 'right').echo, '');
});

test('applyLineEdit: Home 跳到行首（全角宽度）', () => {
  const r = applyLineEdit(st('A【', 2), 'home');
  assert.equal(r.pos, 0);
  assert.equal(r.echo, '\b\b\b');
});

test('applyLineEdit: 行首 Home 无效', () => {
  assert.equal(applyLineEdit(st('AB', 0), 'home').echo, '');
});

test('applyLineEdit: End 跳到行尾', () => {
  const r = applyLineEdit(st('A【', 0), 'end');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, 'A【');
});

test('applyLineEdit: 行尾 End 无效', () => {
  assert.equal(applyLineEdit(st('AB', 2), 'end').echo, '');
});

test('applyLineEdit: Home 后输入插到行首', () => {
  let r = applyLineEdit(st('AB', 2), 'home');
  r = applyLineEdit({ chars: r.chars, pos: r.pos }, 'insert', 'C');
  assert.equal(r.chars.join(''), 'CAB');
  assert.equal(r.pos, 1);
  assert.equal(r.echo, 'CAB\b\b');
});

test('applyLineEdit: secret 模式 Home/End 按 1 列/字符', () => {
  const home = applyLineEdit(st('A【', 2), 'home', '', true);
  assert.equal(home.echo, '\b\b');
  const end = applyLineEdit({ chars: home.chars, pos: home.pos }, 'end', '', true);
  assert.equal(end.pos, 2);
  assert.equal(end.echo, '**');
});

test('applyLineEdit: ← 后插入（模拟方向键+输入）', () => {
  let r = applyLineEdit(st('AB', 2), 'left');
  r = applyLineEdit({ chars: r.chars, pos: r.pos }, 'insert', 'C');
  assert.equal(r.chars.join(''), 'ACB');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, 'CB\b');
});

test('applyLineEdit: secret 模式每字符 1 列回显 *', () => {
  const ins = applyLineEdit(st('AB', 1), 'insert', 'C', true);
  assert.equal(ins.chars.join(''), 'ACB');
  assert.equal(ins.echo, '**\b');
  const bs = applyLineEdit(st('ACB', 2), 'backspace', '', true);
  assert.equal(bs.chars.join(''), 'AB');
  assert.equal(bs.echo, '\b* \b\b');
});

test('applyLineEdit: 代理对字符按码点整删（不留孤立代理项）', () => {
  const r = applyLineEdit(st('a🚀', 2), 'backspace');
  assert.equal(r.chars.join(''), 'a');
  assert.equal(r.echo, '\b\b  \b\b'); // 🚀 显示宽 2
});

test('applyLineEdit: 组合符 0 列——← 跨过无位移', () => {
  const r = applyLineEdit(st('e\u0301', 2), 'left');
  assert.equal(r.pos, 1);
  assert.equal(r.echo, '');
});

test('applyLineEdit: 退格删组合符只改值，不擦基础字符（屏上残影留给下次重绘）', () => {
  const r = applyLineEdit(st('e\u0301', 2), 'backspace');
  assert.equal(r.chars.join(''), 'e');
  assert.equal(r.echo, '');
});

test('applyLineEdit: 基础字符与组合符之间插入', () => {
  const r = applyLineEdit(st('e\u0301', 1), 'insert', 'x');
  assert.equal(r.chars.join(''), 'ex\u0301');
  assert.equal(r.pos, 2);
  assert.equal(r.echo, 'x\u0301');
});

test('applyLineEdit: secret 模式组合符仍按 1 列处理', () => {
  const r = applyLineEdit(st('e\u0301', 2), 'backspace', '', true);
  assert.equal(r.chars.join(''), 'e');
  assert.equal(r.echo, '\b \b');
});
