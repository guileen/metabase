import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';
import { resolve } from 'path';

export default defineConfig(({ mode }) => {
	// Load environment variables from parent directory (root)
	const env = loadEnv(mode, resolve('..'), '');

	return {
		plugins: [sveltekit()],
		server: {
			port: env.SVELTE_DEV_PORT ? parseInt(env.SVELTE_DEV_PORT) : 5173,
			strictPort: false,
			host: true
		},
		build: {
			minify: 'esbuild'
		},
		logLevel: 'error',
		// Environment variables are automatically available via import.meta.env.VITE_*
		// No need for define mapping since we use VITE_ prefixed variables
	};
});