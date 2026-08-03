/**
 * models.ts 单测：档位附加规则矩阵（镜像 Go 版 TestCanAttach1M / TestApplyModelSelection）、
 * fetchModels 请求头与重定向拒绝（本地 http server 断言实际收到的头）、parseModels 解析。
 */
import assert from 'node:assert/strict';
import { createServer, type Server } from 'node:http';
import { test } from 'node:test';

import { applyModelSelection, canAttach1M, fetchModels, parseModels } from './models.js';

test('canAttach1M：opus/sonnet/fable 允许，haiku 不允许', () => {
  assert.equal(canAttach1M('opus'), true);
  assert.equal(canAttach1M('sonnet'), true);
  assert.equal(canAttach1M('fable'), true);
  assert.equal(canAttach1M('haiku'), false);
});

test('applyModelSelection：4 档 × 支持/不支持 + 已带后缀幂等', () => {
  const cases: Array<{ name: string; slot: 'opus' | 'sonnet' | 'haiku' | 'fable'; id: string; hit: boolean; want: string }> = [
    { name: 'opus 命中', slot: 'opus', id: 'glm-5.2', hit: true, want: 'glm-5.2[1m]' },
    { name: 'opus 未命中', slot: 'opus', id: 'glm-5.2-air', hit: false, want: 'glm-5.2-air' },
    { name: 'sonnet 命中', slot: 'sonnet', id: 'deepseek-v4-pro', hit: true, want: 'deepseek-v4-pro[1m]' },
    { name: 'sonnet 未命中', slot: 'sonnet', id: 'deepseek-v4-flash', hit: false, want: 'deepseek-v4-flash' },
    { name: 'haiku 命中也不附加', slot: 'haiku', id: 'glm-5.2', hit: true, want: 'glm-5.2' },
    { name: 'fable 命中附加', slot: 'fable', id: 'glm-5.2', hit: true, want: 'glm-5.2[1m]' },
    { name: 'fable 未命中不附加', slot: 'fable', id: 'glm-5.2-air', hit: false, want: 'glm-5.2-air' },
    { name: '已带后缀幂等', slot: 'opus', id: 'glm-5.2[1m]', hit: true, want: 'glm-5.2[1m]' },
  ];
  for (const c of cases) {
    assert.equal(applyModelSelection(c.slot, c.id, c.hit), c.want, c.name);
  }
});

/** 起一个本地 http server，捕获请求头后按 handler 响应，返回 [url, 断言函数]。 */
function captureServer(
  handler: (req: { headers: Record<string, string | string[] | undefined>; url: string }) => {
    status: number;
    body: string;
    headers?: Record<string, string>;
  },
): Promise<{ url: string; server: Server; close: () => Promise<void> }> {
  return new Promise((resolve) => {
    const server = createServer((req, res) => {
      const { status, body, headers } = handler({
        headers: req.headers as Record<string, string | string[] | undefined>,
        url: req.url ?? '',
      });
      res.writeHead(status, { 'content-type': 'application/json', ...headers });
      res.end(body);
    });
    server.listen(0, '127.0.0.1', () => {
      const addr = server.address();
      if (!addr || typeof addr === 'string') throw new Error('无监听地址');
      resolve({
        url: `http://127.0.0.1:${addr.port}`,
        server,
        close: () => new Promise((r) => server.close(() => r())),
      });
    });
  });
}

async function withServer(
  handler: (req: { headers: Record<string, string | string[] | undefined>; url: string }) => {
    status: number;
    body: string;
    headers?: Record<string, string>;
  },
  fn: (url: string) => Promise<void>,
): Promise<void> {
  const s = await captureServer(handler);
  try {
    await fn(s.url);
  } finally {
    await s.close();
  }
}

test('fetchModels：AUTH_TOKEN 只发 Authorization + anthropic-version', async () => {
  await withServer((req) => {
    const h = req.headers;
    assert.equal(h['authorization'], 'Bearer t');
    assert.equal(h['x-api-key'], undefined, 'AUTH_TOKEN 模式不应带 x-api-key');
    assert.equal(h['anthropic-version'], '2023-06-01');
    return { status: 200, body: '{"data":[{"id":"glm-5.2"}]}' };
  }, async (url) => {
    const models = await fetchModels(url, 't', 'AUTH_TOKEN');
    assert.equal(models[0]?.id, 'glm-5.2');
  });
});

test('fetchModels：API_KEY 只发 x-api-key', async () => {
  await withServer((req) => {
    const h = req.headers;
    assert.equal(h['x-api-key'], 'k');
    assert.equal(h['authorization'], undefined, 'API_KEY 模式不应带 Authorization');
    return { status: 200, body: '{"data":[{"id":"m"}]}' };
  }, async (url) => {
    await fetchModels(url, 'k', 'API_KEY');
  });
});

test('fetchModels：302 重定向拒绝且报错带地址', async () => {
  await withServer(
    () => ({ status: 302, body: '', headers: { location: 'https://example.com/other' } }),
    async (url) => {
      await assert.rejects(
        fetchModels(url, 'k', 'API_KEY'),
        /重定向.*https:\/\/example\.com\/other/,
      );
    },
  );
});

test('fetchModels：响应超过 1 MiB 拒绝', async () => {
  await withServer(
    () => ({ status: 200, body: 'x'.repeat(2 << 20) }),
    async (url) => {
      await assert.rejects(fetchModels(url, 'k', 'API_KEY'), /超过 1 MiB/);
    },
  );
});

test('fetchModels：端点推导 {base}/v1/models', async () => {
  let seen = '';
  await withServer((req) => {
    seen = req.url;
    return { status: 200, body: '{"data":[{"id":"m"}]}' };
  }, async (url) => {
    await fetchModels(`${url}/anthropic`, 'k', 'API_KEY');
    assert.equal(seen, '/anthropic/v1/models');
  });
});

test('parseModels：Anthropic 风格', () => {
  const body = '{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5"},{"id":"claude-opus-4-8"}]}';
  assert.deepEqual(parseModels(body), [
    { id: 'claude-sonnet-4-5', displayName: 'Claude Sonnet 4.5' },
    { id: 'claude-opus-4-8' },
  ]);
});

test('parseModels：OpenAI 风格', () => {
  const body = '{"object":"list","data":[{"id":"mimo-v2.5-pro","object":"model"},{"id":"deepseek-chat"}]}';
  assert.deepEqual(parseModels(body), [{ id: 'mimo-v2.5-pro' }, { id: 'deepseek-chat' }]);
});

test('parseModels：HTTP 200 业务错误体报错', () => {
  assert.throws(
    () => parseModels('{"code":1001,"msg":"Header中未收到Authorization参数，无法进行身份验证。","success":false}'),
    /Authorization/,
  );
});

test('parseModels：空列表报错', () => {
  assert.throws(() => parseModels('{"data":[]}'), /模型列表为空/);
});
