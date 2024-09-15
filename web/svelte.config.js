import adapter from "@megakuul/adapter-battleshiper";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter(),
	}
};

export default config;
