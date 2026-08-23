/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-auto';
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

			// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
			// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
			// See https://github.com/sveltejs/vite-plugin-svelte/blob/main/docs/faq.md#defining-vite-config-from-the-svelte-config
			adapter: adapter()
		})
	],
	test: {
		// The vendored OpenMouse reference under openmouse-dev/ carries its own
		// test suite; only this app's tests run here.
		exclude: ['**/node_modules/**', 'openmouse-dev/**']
	}
});
