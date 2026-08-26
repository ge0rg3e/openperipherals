// Packs installers with the version from the repo-root package.json - the
// single source of truth for releases - by injecting it into electron-builder
// as extra metadata instead of reading the workspace package.json.
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const rootPackage = path.resolve(here, '..', '..', '..', 'package.json');
const { version } = JSON.parse(readFileSync(rootPackage, 'utf8'));
if (!version) {
	console.error(`No version found in ${rootPackage}`);
	process.exit(1);
}

console.log(`Building OpenPeripherals installers v${version}`);
const result = spawnSync(
	'electron-builder',
	['--publish', 'never', '--config', `extraMetadata.version=${version}`],
	{ stdio: 'inherit', shell: process.platform === 'win32' }
);
process.exit(result.status ?? 1);
