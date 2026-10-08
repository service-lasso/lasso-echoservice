import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import http from 'node:http';
import { prepareEchoWebDAVService } from './prepare.mjs';
import { checkEchoSecretFile } from './check.mjs';

test('preparation produces a complete discoverable service and refuses overwriting it', async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'echo-reference-'));
  try {
    const target = await prepareEchoWebDAVService(root, { build: output => writeFile(output, 'synthetic test binary'), platform: 'linux' });
    const manifest = JSON.parse(await readFile(path.join(target, 'service.json')));
    assert.equal(manifest.executable, './echo-secret-demo');
    assert.deepEqual(manifest.args, []);
    assert.equal(manifest.env.ECHO_SECRET_FILES_DIR, '${SERVICE_LASSO_SECRETS_DIR}');
    assert.equal(manifest.broker.imports[0].required, true);
    assert.equal(manifest.config, undefined);
    assert.deepEqual(manifest.broker.files, [{ path: "demo-config.json", content: '{"demoCredential":"${echo.DEMO_CREDENTIAL}"}' }]);
    assert.equal(manifest.endpoints.filter(e => e.kind === 'network').length, 3);
    await assert.rejects(prepareEchoWebDAVService(root), { code: 'EEXIST' });
    assert.equal(await readFile(path.join(target, 'echo-secret-demo'), 'utf8'), 'synthetic test binary');
  } finally { await rm(root, { recursive: true, force: true }); }
});

test('checker proves a successful reread and rejects unavailable or remote status', async () => {
  let reads = 1, available = true, oversized = false, advance = true;
  const server = http.createServer((req, res) => {
    if (req.method === 'POST' && advance) reads++;
    res.setHeader('content-type', 'application/json');
    res.end(oversized ? 'x'.repeat(8192) : JSON.stringify({ status: available ? 'loaded' : 'unavailable', sizeBytes: 48, reads, privateValue: 'must never be returned by checker' }));
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  try {
    const origin = `http://127.0.0.1:${server.address().port}`;
    assert.deepEqual(await checkEchoSecretFile(origin), { status: 'loaded', sizeBytes: 48, reads: 2 });
    available = false;
    await assert.rejects(checkEchoSecretFile(origin), /has not loaded/);
    available = true; advance = false;
    await assert.rejects(checkEchoSecretFile(origin), /did not advance/);
    oversized = true;
    await assert.rejects(checkEchoSecretFile(origin), /exceeds limit/);
    for (const url of ['http://example.com:8080', 'http://user:pass@127.0.0.1:4010', origin + '/private', origin + '?token=private']) {
      await assert.rejects(checkEchoSecretFile(url), /loopback/);
    }
  } finally { await new Promise(resolve => server.close(resolve)); }
});
