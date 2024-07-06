import { existsSync, writeFileSync } from 'node:fs';

/** @param {import('./index.js').default} */
export default function (options = {}) {
	/** @type {import('@sveltejs/kit').Adapter} */
	const adapter = {
		name: '@megakuul/adapter-battleshiper',
		async adapt(builder) {

      const dest = builder.getBuildDirectory("battleshiper")
      const tmp = builder.getBuildDirectory("battleshiper-tmp")

      builder.rimraf(dest)
      builder.rimraf(tmp)

      builder.mkdirp(dest)
      builder.mkdirp(tmp)

      writeFileSync(
        `${tmp}/manifest.json`,
        [
          `export const manifest = ${builder.generateManifest({ relativePath: "./" })};`,
          `export const prerendered = new Set(${JSON.stringify(builder.prerendered.paths)});`,
          `export const base = ${JSON.stringify(builder.config.kit.paths.base)};`,
        ].join("\n")
      )

      
		},
	};

	return adapter;
}