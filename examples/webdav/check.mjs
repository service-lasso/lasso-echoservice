import path from 'node:path';
import { pathToFileURL } from 'node:url';

export async function checkEchoSecretFile(origin) {
  const url = new URL(origin);
  if (url.protocol !== 'http:' || url.hostname !== '127.0.0.1' || !url.port || url.username || url.password || url.pathname !== '/' || url.search || url.hash) {
    throw new Error('Use the Echo loopback HTTP origin shown in Service Admin, e.g. http://127.0.0.1:4010');
  }
  url.pathname = '/secret-file';
  async function read(method) {
    const response = await fetch(url, { method, redirect: 'error', signal: AbortSignal.timeout(3000) });
    if (!response.ok) throw new Error('Echo secret-file status request failed');
    const reader = response.body.getReader();
    const chunks = []; let size = 0;
    try {
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        size += value.byteLength;
        if (size > 4096) throw new Error('Echo status exceeds limit');
        chunks.push(value);
      }
    } finally { await reader.cancel(); }
    const text = Buffer.concat(chunks).toString('utf8');
    const status = JSON.parse(text);
    if (status.status !== 'loaded' || !Number.isSafeInteger(status.sizeBytes) || status.sizeBytes <= 0 || !Number.isSafeInteger(status.reads) || status.reads < 1) {
      throw new Error('Echo has not loaded a valid Broker file');
    }
    return { status: status.status, sizeBytes: status.sizeBytes, reads: status.reads };
  }
  const before = await read('GET');
  const after = await read('POST');
  if (after.reads !== before.reads + 1 || after.sizeBytes !== before.sizeBytes) throw new Error('Echo reread did not advance its counter');
  return after;
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    if (process.argv.length !== 3) throw new Error('Usage: node examples/webdav/check.mjs <echo-http-origin>');
    console.log(JSON.stringify(await checkEchoSecretFile(process.argv[2]), null, 2));
  } catch { console.error('Secret-file check failed. Verify the loopback Echo origin and its Broker provisioning status.'); process.exitCode = 1; }
}
