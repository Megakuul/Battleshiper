<script>
  import { cn } from "$lib/utils";

	/** @type {number} */
	export let gridBlockSize = 32;
	/** @type {string} */
	export let gridColor = "white";
	/** @type {boolean} */
	export let showFade = true;
	/** @type {string} */
	export let className = "";

	const GRIDSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="${gridBlockSize}" height="${gridBlockSize}" fill="none" stroke="${gridColor}"><path d="M0 .5H31.5V32"/></svg>`
</script>

<div
	style="background-image: url('data:image/svg+xml;utf8,{GRIDSVG}');"
	class={cn("relative w-full bg-black bg-opacity-20", className)}
>
	{#if showFade}
		<div
			class="pointer-events-none absolute inset-0 flex items-center justify-center bg-black [mask-image:radial-gradient(ellipse_at_center,transparent_20%,black)] "
		></div>
	{/if}
	<slot />
</div>

<style>
</style>