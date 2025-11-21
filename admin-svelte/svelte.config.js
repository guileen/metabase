import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: '../web/admin',
			assets: '../web/admin',
			fallback: 'index.html',
			precompress: false,
			strict: true
		})
	}
};

export default config;