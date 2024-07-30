<script lang="ts">
	import { cn } from '$lib/utils';

  // relative to top-0 & left-0 (which means that the svg is centered)

  /** @type {number} */
  export let srcPercentX;
  /** @type {number} */
  export let srcPercentY;
  /** @type {number} */
  export let destPercentX; 
  /** @type {number} */
  export let destPercentY;
  /** @type {number} */
  export let rotationDeg;


  /** @type {string} */
	export let className;
  /** @type {string} */
	export let fill = "white";
</script>

<svg
  style="--x-source: {srcPercentX}%; --y-source: {srcPercentY}%; --x-destination: {destPercentX}%; --y-destination: {destPercentY}%; --rotate-destination: {rotationDeg}deg;"
	class={cn("spotlight absolute top-0 left-0 opacity-0 pointer-events-none z-0 h-[169%] w-[138%] lg:w-[84%]", className)}
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 3787 2842"
	fill="none">

	<g filter="url(#filter)">
		<ellipse
			cx="1924.71"
			cy="273.501"
			rx="1924.71"
			ry="273.501"
			transform="matrix(-0.822377 -0.568943 -0.568943 0.822377 3631.88 2291.09)"
			fill={fill}
			fill-opacity="0.21"
		></ellipse>
	</g>
	<defs>
		<filter
			id="filter"
			x="0.860352"
			y="0.838989"
			width="3785.16"
			height="2840.26"
			filterUnits="userSpaceOnUse"
			color-interpolation-filters="sRGB"
		>
			<feFlood flood-opacity="0" result="BackgroundImageFix"></feFlood>
			<feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape"></feBlend>
			<feGaussianBlur stdDeviation="151" result="effect1_foregroundBlur_1065_8"></feGaussianBlur>
		</filter>
	</defs>
</svg>


<style>
  .spotlight {
    rotate: var(--rotate-destination);
    animation: spotlight 2s ease .75s 1 forwards;
  }

  @keyframes spotlight {
    0% {
      opacity: 0;
      transform: translate(var(--x-source), var(--y-source));
      scale: 0.5;
    }
    100% {
      opacity: 1;
      transform: translate(var(--x-destination), var(--y-destination));
      scale: 1;
    }
  }
</style>