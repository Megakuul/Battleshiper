import adapter from "@megakuul/adapter-battleshiper";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			debug: true,
		}),
	}
};

export default config;
