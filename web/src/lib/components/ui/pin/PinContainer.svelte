<script>
	import { cn } from '$lib/utils';
	import PinPerspective from './PinPerspective.svelte';

	/** @type {string | undefined} */
	export let title;
	/** @type {string | undefined} */
	export let href;
	/** @type {string | undefined} */
	export let className = undefined;
	/** @type {string | undefined} */
	export let containerClassName = undefined;

	let transform = 'translate(-50%,-50%) rotateX(0deg)';

	const onMouseEnter = () => {
		transform = 'translate(-50%,-50%) rotateX(40deg) scale(0.8)';
	};
	const onMouseLeave = () => {
		transform = 'translate(-50%,-50%) rotateX(0deg) scale(1)';
	};
</script>

<div
	role="button"
	tabindex="0"
	class={cn('group/pin relative cursor-pointer w-max', containerClassName)}
	on:mouseenter={onMouseEnter}
	on:mouseleave={onMouseLeave}
>
	<div
		style="perspective: 1000px; translateZ(0);"
		class="absolute left-1/2 top-1/2 ml-[0.09375rem] mt-4 -translate-x-1/2 -translate-y-1/2"
	>
		<div
			style={`transform: ${transform};`}
			class="absolute left-1/2 top-1/2 flex items-start justify-start overflow-hidden rounded-2xl border border-white/[0.1] bg-black p-4 shadow-[0_8px_16px_rgb(0_0_0/0.4)] transition duration-700 group-hover/pin:border-white/[0.2]"
		>
			<div class={cn('relative z-50', className)}><slot /></div>
		</div>
	</div>
	<PinPerspective {title} {href} />
</div>