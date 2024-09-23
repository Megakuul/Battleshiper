import { writeFileSync } from 'node:fs';
import esbuild from 'esbuild';
import { fileURLToPath } from 'node:url';
import { posix } from 'node:path';
import AdmZip from "adm-zip"; 

/** @param {import('./index.js').default} */
export default function (options) {
	/** @type {import('@sveltejs/kit').Adapter} */
	const adapter = {
		name: '@megakuul/adapter-battleshiper',
		async adapt(builder) {

      const src = fileURLToPath(new URL('./src', import.meta.url).href);
      const dest = "build"
      const tmp = builder.getBuildDirectory("battleshiper")

      builder.rimraf(dest)
      builder.rimraf(tmp)

      builder.mkdirp(dest)
      builder.mkdirp(tmp)

      writeFileSync(
        `${tmp}/manifest.js`,
        [
          `export const manifest = ${builder.generateManifest({ relativePath: posix.relative(tmp, builder.getServerDirectory()) })};`,
          `export const prerendered = new Set(${JSON.stringify(builder.prerendered.paths)});`,
          `export const base = ${JSON.stringify(builder.config.kit.paths.base)};`,
        ].join("\n")
      )

      builder.writeClient(`${dest}/client`)
      builder.writePrerendered(`${dest}/prerendered`);

      builder.copy(`${src}/lambda-handler.js`, `${tmp}/index.js`, {
        replace: {
          SERVER: `${builder.getServerDirectory()}/index.js`,
          SHIMS: `${src}/shims.js`,
          MANIFEST: `${tmp}/manifest.js`,
          DEBUGMODE: options?.debug ? true : false,
        }
      })

      try {
        const result = await esbuild.build({
          entryPoints: [`${tmp}/index.js`],
          outfile: `${dest}/server/index.js`,
          platform: "node",
          target: "node20",
          format: "esm",
          bundle: true,
          sourcemap: "linked",
        })
        if (result.warnings.length > 0) {
          console.error((await esbuild.formatMessages(result.warnings, {
            kind: "warning",
            color: true,
          })).join("\n"))
          return
        }
      } catch (err) {
        const error = /** @type {import('esbuild').BuildFailure} */ (err);
        console.error((await esbuild.formatMessages(error.errors, {
          kind: "error",
          color: true,
        })).join("\n"))
        return
      }

      try {
        const zip = new AdmZip();
        zip.addLocalFile(`${dest}/server/index.js`, "", "index.mjs");
        await zip.writeZipPromise(`${dest}/server/handler.zip`);
      } catch (err) {
        console.error("failed to create zip file:")
        console.error(err.message)
      }
		},
	};

	return adapter;
}