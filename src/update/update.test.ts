import { test } from 'node:test';
import assert from 'node:assert/strict';

import { parseSemver, upgradeCommand } from './update.js';

test('parseSemver 解析标准/带 v/带后缀版本', () => {
  assert.deepEqual(parseSemver('0.4.25'), [0, 4, 25]);
  assert.deepEqual(parseSemver('v0.4.25'), [0, 4, 25]);
  assert.deepEqual(parseSemver('0.4.25-rc1'), [0, 4, 25]);
  assert.deepEqual(parseSemver('1.2.3'), [1, 2, 3]);
  assert.equal(parseSemver('dev'), undefined);
  assert.equal(parseSemver(''), undefined);
  assert.equal(parseSemver('garbage'), undefined);
});

test('upgradeCommand 按版本门控：>= 0.4.25 用 xx update，旧版用 npm 命令', () => {
  const npm = 'npm i -g @cc-x/cc-x@latest';
  assert.equal(upgradeCommand('0.4.24'), npm);
  assert.equal(upgradeCommand('0.4.24-rc1'), npm);
  assert.equal(upgradeCommand('0.4.25'), 'xx update');
  assert.equal(upgradeCommand('0.4.25-rc1'), 'xx update');
  assert.equal(upgradeCommand('0.4.26'), 'xx update');
  assert.equal(upgradeCommand('v0.4.25'), 'xx update');
  assert.equal(upgradeCommand('1.0.0'), 'xx update');
  assert.equal(upgradeCommand('dev'), npm); // 解析失败不误报
});
