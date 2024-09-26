# adapter-battleshiper

A SvelteKit adapter for deployment on battleshiper lambda infrastructure.


## Installation

Install the adapter and update your `svelte.config.js`:

```bash
# Installing from npm
npm i --save-dev @megakuul/adapter-battleshiper
# Or even better: install from jsr
npx jsr add --save-dev @megakuul/adapter-battleshiper
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