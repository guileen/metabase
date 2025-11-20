import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: '../web/admin-svelte',
			assets: '../web/admin-svelte',
			fallback: 'index.html',
			precompress: false,
			strict: true
		})
	}
};

export default config;