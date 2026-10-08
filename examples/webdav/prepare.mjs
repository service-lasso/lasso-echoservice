import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { spawn } from 'node:child_process';

const sourceRoot = fileURLToPath(new URL('../../', import.meta.url));

async function buildEcho(output) {
  await new Promise((resolve, reject) => {
    const child = spawn('go', ['build', '-o', output, '.'], { cwd: sourceRoot, stdio: 'inherit' });
    child.once('error', reject);
    child.once('exit', code => code === 0 ? resolve() : reject(new Error('Echo build failed')));
  });
}

export async function prepareEchoWebDAVService(servicesRoot, { build = buildEcho, platform = process.env.GOOS || process.platform } = {}) {
  const root = path.resolve(servicesRoot);
  if (!(await stat(root)).isDirectory()) throw new Error('Services root must be an existing directory');
  const target = path.join(root, 'echo-webdav');
  // Refuse an existing service, including a symlink. Never overwrite its files.
  await mkdir(target);
  const binary = platform === 'win32' || platform === 'windows' ? 'echo-secret-demo.exe' : 'echo-secret-demo';
  await build(path.join(target, binary));
  const manifest = JSON.parse(await readFile(new URL('./service.json', import.meta.url), 'utf8'));
  manifest.executable = './' + binary;
  manifest.args = [];
  await writeFile(path.join(target, 'service.json'), JSON.stringify(manifest, null, 2) + '\n', { flag: 'wx' });
  return target;
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try {
    if (process.argv.length !== 3) throw new Error('Usage: node examples/webdav/prepare.mjs <existing-services-root>');
    console.log('Prepared service:', await prepareEchoWebDAVService(process.argv[2]));
    console.log('Create shared/echo/echo.DEMO_CREDENTIAL in Broker, then discover, install, configure and start echo-webdav.');
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
