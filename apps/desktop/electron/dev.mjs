// Dev launcher: starts the web app's Vite dev server, then boots Electron
// pointed at it. Lives in the desktop package but serves apps/web.
import { createServer } from 'vite';
import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', 'web');

const server = await createServer({
	root: webRoot,
	configFile: path.join(webRoot, 'vite.config.ts')
});
await server.listen();

const url = server.resolvedUrls.local[0];
console.log(`vite dev server ready at ${url}, starting electron...`);

const electron = spawn('electron', ['.'], {
	stdio: 'inherit',
	env: { ...process.env, VITE_DEV_SERVER_URL: url },
	shell: process.platform === 'win32'
});

electron.on('close', async () => {
	await server.close();
	process.exit(0);
});

for (const signal of ['SIGINT', 'SIGTERM']) {
	process.on(signal, () => {
		electron.kill(signal);
	});
}
