import assert from 'node:assert/strict';
import test from 'node:test';

import { scrollWindow } from './select.js';

// 滚动窗口单测（镜像 Go 版 TestScrollWindow）：窗口始终包含选中项，且向 [0,n] 内收敛不越界。
test('scrollWindow：全部能装下不滚动', () => {
  assert.deepEqual(scrollWindow(0, 0, 5, 3), { start: 0, end: 3 });
});
test('scrollWindow：选中项在顶部窗口内', () => {
  assert.deepEqual(scrollWindow(0, 3, 5, 20), { start: 0, end: 5 });
});
test('scrollWindow：中部下移滑窗', () => {
  assert.deepEqual(scrollWindow(0, 12, 5, 20), { start: 8, end: 13 });
});
test('scrollWindow：底部钳制', () => {
  assert.deepEqual(scrollWindow(0, 19, 5, 20), { start: 15, end: 20 });
});
test('scrollWindow：上移超过顶界', () => {
  assert.deepEqual(scrollWindow(10, 2, 5, 20), { start: 2, end: 7 });
});
test('scrollWindow：viewH 最小为 1', () => {
  assert.deepEqual(scrollWindow(0, 5, 1, 20), { start: 5, end: 6 });
});
test('scrollWindow：n 小于 viewH 全量显示', () => {
  assert.deepEqual(scrollWindow(0, 0, 10, 4), { start: 0, end: 4 });
});
test('scrollWindow：任意输入下选中项必可见', () => {
  for (const n of [4, 20, 100]) {
    for (const viewH of [1, 5, 25]) {
      for (let idx = 0; idx < n; idx++) {
        const { start, end } = scrollWindow(50, idx, viewH, n);
        assert.ok(idx >= start && idx < end, `n=${n} viewH=${viewH} idx=${idx} → [${start},${end}) 未包含选中项`);
      }
    }
  }
});
