/// <reference types="vitest/config" />
import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	// The repo-root package.json is the version source of truth; the landing
	// page imports it for the download URLs. An alias is required because
	// relative imports may not leave the Vite project root.
	resolve: {
		alias: {
			'@root/package.json': fileURLToPath(new URL('../../package.json', import.meta.url))
		}
	},
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// adapter-static emits a fully static site (prerendered landing page +
			// SPA fallback for the client-only /app route) which the Electron
			// main process serves via a custom app:// protocol.
			adapter: adapter({ fallback: 'app.html' })
		})
	],
	test: {
		// The vendored OpenMouse reference under openmouse-dev/ carries its own
		// test suite; only this app's tests run here.
		exclude: ['**/node_modules/**', 'openmouse-dev/**']
	}
});
