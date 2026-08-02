/**
 * markOnboardingDoneIn 表驱动测试：镜像 Go 版 internal/claudecfg/onboarding_write_test.go。
 * 断言全部走逐字节 Buffer 比对（readFileSync 不带 encoding）。
 */
import assert from 'node:assert/strict';
import { chmodSync, mkdtempSync, readFileSync, statSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';

import { markOnboardingDoneIn } from './claudecfg.js';

// 顶层循环注册（不用嵌套 test()：本 Node 版本嵌套注册在父级同步返回时会取消子测试）。
// 每个用例独立 temp dir，互不干扰。
function runCases(cases: Array<{ name: string; in: string; want?: string }>): void {
  for (const tc of cases) {
    test(tc.name, () => {
      const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
      const p = join(dir, '.claude.json');
      writeFileSync(p, tc.in);
      const err = markOnboardingDoneIn(p);
      const got = readFileSync(p);
      if (tc.want === undefined) {
        assert.ok(err, `期望报错，却成功; 结果: ${got.toString()}`);
        assert.equal(got.toString(), tc.in, '报错时文件被改动');
        return;
      }
      assert.equal(err, undefined, `写入失败: ${err}`);
      assert.equal(got.toString(), tc.want, '结果不符');
    });
  }
}

runCases([
  { name: '替换 false', in: '{"a":1,"hasCompletedOnboarding":false,"b":2}', want: '{"a":1,"hasCompletedOnboarding":true,"b":2}' },
  { name: '键与值之间空白保留', in: '{ "hasCompletedOnboarding" : false , "x" : 1 }', want: '{ "hasCompletedOnboarding" : true , "x" : 1 }' },
  { name: '值替换全谱：null', in: '{"hasCompletedOnboarding":null,"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '值替换全谱：字符串', in: '{"hasCompletedOnboarding":"pending","a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '值替换全谱：数字', in: '{"hasCompletedOnboarding":123,"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '值替换全谱：负数指数', in: '{"hasCompletedOnboarding":-1.5e3,"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '值替换全谱：对象', in: '{"hasCompletedOnboarding":{"x":1},"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '值替换全谱：数组', in: '{"hasCompletedOnboarding":[1,2],"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
  { name: '空对象插入不带尾逗号', in: '{}', want: '{"hasCompletedOnboarding": true}' },
  { name: '空对象带空白', in: '{  }', want: '{"hasCompletedOnboarding": true  }' },
  { name: '顶层无键插入', in: '{"a":1}', want: '{"hasCompletedOnboarding": true,"a":1}' },
  { name: '插入不破坏原格式', in: '{ "a" : 1 }', want: '{"hasCompletedOnboarding": true, "a" : 1 }' },
  { name: '字符串值内同名文本不误命中', in: '{"note":"hasCompletedOnboarding","a":1}', want: '{"hasCompletedOnboarding": true,"note":"hasCompletedOnboarding","a":1}' },
  { name: '嵌套对象同名键不动', in: '{"mcp":{"hasCompletedOnboarding":false},"a":1}', want: '{"hasCompletedOnboarding": true,"mcp":{"hasCompletedOnboarding":false},"a":1}' },
  { name: '重复顶层键取最后一个', in: '{"hasCompletedOnboarding":false,"hasCompletedOnboarding":false,"a":1}', want: '{"hasCompletedOnboarding":false,"hasCompletedOnboarding":true,"a":1}' },
  { name: '中文与 emoji 环绕', in: '{"项目":"计划 😀","hasCompletedOnboarding":false,"备注":"含 UTF-8 中文"}', want: '{"项目":"计划 😀","hasCompletedOnboarding":true,"备注":"含 UTF-8 中文"}' },
  { name: '键在文件尾部（大前置）', in: '{' + `"filler":"x",`.repeat(50000) + '"hasCompletedOnboarding":false}', want: '{' + `"filler":"x",`.repeat(50000) + '"hasCompletedOnboarding":true}' },
  { name: '已是 true 原样返回', in: '{"hasCompletedOnboarding":true,"a":1}', want: '{"hasCompletedOnboarding":true,"a":1}' },
]);

runCases([
  { name: '残缺 JSON', in: '{' },
  { name: '空文件', in: '' },
  { name: '顶层数组', in: '[1,2,3]' },
  { name: '顶层字符串', in: '"hello"' },
  { name: '顶层数字', in: '42' },
  { name: 'BOM 前缀', in: '﻿{"hasCompletedOnboarding":false}' },
]);

test('markOnboardingDone 文件不存在时创建最小文件', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  const p = join(dir, '.claude.json');
  assert.equal(markOnboardingDoneIn(p), undefined);
  assert.equal(readFileSync(p).toString(), '{"hasCompletedOnboarding": true}\n');
});

// 镜像 Go 版 TestMarkOnboardingDoneBOMInsideAfterValidPrefix：BOM（U+FEFF）在字符串值内部
// 是合法 JSON，splice 必须按字节处理、不污染多字节内容。
test('markOnboardingDone 字符串值内嵌 BOM 不被污染', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  const p = join(dir, '.claude.json');
  const bom = Buffer.from([0xef, 0xbb, 0xbf]);
  writeFileSync(p, Buffer.concat([Buffer.from('{"x":"'), bom, Buffer.from('","hasCompletedOnboarding":false}')]));
  assert.equal(markOnboardingDoneIn(p), undefined);
  const got = readFileSync(p);
  const want = Buffer.concat([Buffer.from('{"x":"'), bom, Buffer.from('","hasCompletedOnboarding":true}')]);
  assert.equal(got.equals(want), true, `多字节内容被污染: ${got.toString('hex')}`);
});

test('markOnboardingDone 幂等：第二次不写（字节与 mtime 不变）', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  const p = join(dir, '.claude.json');
  assert.equal(markOnboardingDoneIn(p), undefined);
  const before = readFileSync(p);
  const st1 = statSync(p);
  assert.equal(markOnboardingDoneIn(p), undefined);
  const after = readFileSync(p);
  const st2 = statSync(p);
  assert.equal(before.toString(), after.toString(), '第二次调用改了文件');
  assert.equal(st1.mtimeMs, st2.mtimeMs, '第二次调用改了 mtime');
});

test('markOnboardingDone 路径是目录报错', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  assert.ok(markOnboardingDoneIn(dir), '路径是目录应报错');
});

test('markOnboardingDone 不存在的目录报错', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  assert.ok(markOnboardingDoneIn(join(dir, 'nope', '.claude.json')), '不存在的目录应报错');
});

test('markOnboardingDone 符号链接本体不被替换', { skip: process.platform === 'win32' }, () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  const target = join(dir, 'target.json');
  const link = join(dir, '.claude.json');
  writeFileSync(target, '{"hasCompletedOnboarding":false}');
  symlinkSync(target, link);
  assert.equal(markOnboardingDoneIn(link), undefined);
  assert.equal(readFileSync(target).toString(), '{"hasCompletedOnboarding":true}');
  assert.equal(statSync(link).isSymbolicLink(), true, '符号链接本体被替换成了普通文件');
});

test('markOnboardingDone 只读文件报错且不改', () => {
  const dir = mkdtempSync(join(tmpdir(), 'ccx-claudecfg-'));
  const p = join(dir, '.claude.json');
  writeFileSync(p, '{"hasCompletedOnboarding":false}');
  if (process.platform !== 'win32') chmodSync(p, 0o444); // Windows 上只读位不阻止写
  const err = markOnboardingDoneIn(p);
  if (process.platform !== 'win32') {
    assert.ok(err, '只读文件应报错');
    assert.equal(readFileSync(p).toString(), '{"hasCompletedOnboarding":false}', '只读文件被改动');
  }
});
