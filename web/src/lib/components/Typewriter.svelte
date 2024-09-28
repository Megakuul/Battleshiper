<script>
	import { cn } from '$lib/utils';
  import { onMount } from 'svelte';
	import { Motion } from 'svelte-motion';
    import { fade } from 'svelte/transition';

	/**
	 * @typedef Word
	 * @property {string} word
	 * @property {string} [className]
	*/

	/** 
	 * @typedef Text
	 * @property {Word[]} words
	*/

  /** @type {Word} */
  export let prefix;

	/** @type {Text[]} */
	export let texts;

	/** @type {number} */
	export let duration;

	/** @type {string | undefined} */
	export let className = undefined;
	/** @type {string | undefined} */
	export let cursorClassName = undefined;

	/** @type {"hidden" | "pause" | "visible"} */
	let animationState = "hidden";

	/** @type {Text} */
	let animatedText = {
		words: [{word: ""}],
	};

	/** @type {number} */
	let animatedTextIdx = 0;

	onMount(() => {
		setInterval(() => {
			switch (animationState) {
				case "hidden":
					if (animatedTextIdx >= texts.length) {
						animatedTextIdx = 0;
					}
					
					animatedText = texts[animatedTextIdx];

					animatedTextIdx++;
					animationState = "visible";
					break;
				case "visible":
					animationState = "pause";
					break;	
				case "pause":
					animationState = "hidden";
					break;
			}
		}, duration * 1000)
	})

	const variants = {
		visible: {
			width: "fit-content",
		},
		pause: {
			width: "fit-content",
		},
		hidden: { 
			width: 0 
		}
	};
</script>

<Motion
	let:motion
	initial={{
		width: 0
	}}
	variants={variants}
	animate={animationState}
	transition={{
		duration: duration,
		ease: 'linear',
		delay: 0
	}}>
	<div class="{cn("flex flex-col sm:flex-row gap-0 sm:gap-2 items-center justify-center text-3xl sm:text-5xl xl:text-6xl")}">
		<span class={cn("text-nowrap py-2", prefix.className)}>{prefix.word}</span>
		<div class="flex flex-row gap-1 items-center min-h-10">
			<span use:motion class="overflow-hidden text-nowrap py-2">
				{#each animatedText.words as word}
					<span class="{word.className}">{word.word}{' '}</span>
				{/each}
			</span>
			<Motion
				let:motion
				initial={{
					opacity: 0
				}}
				animate={{
					opacity: 1
				}}
				transition={{
					duration: 0.8,
		
					repeat: Infinity,
					repeatType: 'reverse'
				}}>
				<span
					use:motion
					class={cn("w-[4px] bg-black h-6 sm:h-10 xl:h-12", cursorClassName)}
				></span>
			</Motion>
		</div>
	</div>
</Motion>

<div class={cn('my-6 flex flex-row items-center space-x-2 ', className)}>

</div>