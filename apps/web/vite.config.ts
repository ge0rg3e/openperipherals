/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
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
