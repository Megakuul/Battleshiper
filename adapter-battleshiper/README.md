# adapter-battleshiper

A SvelteKit adapter for deployment on battleshiper lambda infrastructure.


## Installation

Install the adapter and update your `svelte.config.js`:

```bash
npm i @megakuul/adapter-battleshiper --save-dev
```

```javascript
import adapter from "@megakuul/adapter-battleshiper";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter(),
	}
};

export default config;
```