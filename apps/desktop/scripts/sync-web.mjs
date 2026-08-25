// Copies the built web app (apps/web/build) into the desktop package so
// electron-builder can pack it and main.cjs can serve it over app://.
import { cpSync, existsSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const desktopRoot = path.resolve(here, '..');
const src = path.resolve(desktopRoot, '..', 'web', 'build');
const dest = path.join(desktopRoot, 'build');

if (!existsSync(src)) {
	console.error(`No web build found at ${src}. Run "npm run build" for @openperipherals/web first.`);
	process.exit(1);
}

rmSync(dest, { recursive: true, force: true });
cpSync(src, dest, { recursive: true });
console.log(`Synced ${path.relative(process.cwd(), src)} -> ${path.relative(process.cwd(), dest)}`);
