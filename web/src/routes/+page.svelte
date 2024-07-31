<script>
  import Battleship from "$lib/components/Battleship.svelte";
	import GridBackground from "$lib/components/GridBackground.svelte";
  import Spotlight from "$lib/components/Spotlight.svelte";
  import Typewriter from "$lib/components/Typewriter.svelte";

</script>

<svelte:head>
	<title>Home</title>
	<meta name="description" content="Svelte demo app" />
</svelte:head>

<GridBackground gridBlockSize={64} gridColor="rgba(255, 255, 255, 0.2)">
	<Spotlight
		fill="rgba(255, 255, 255, 0.7)"
		srcPercentX="{-90}"
		srcPercentY="{-20}"
		destPercentX="{-60}"
		destPercentY="{-10}"
		rotationDeg="{-45}"
		className="md:left-60 md:-top-20" 
	/>
	<Spotlight
		fill="rgba(255, 255, 255, 0.7)"
		srcPercentX="{90}"
		srcPercentY="{-45}"
		destPercentX="{70}"
		destPercentY="{-40}"
		rotationDeg="{125}"
		className="md:left-60 md:-top-20" 
	/>
	<div class="w-11/12 flex flex-col-reverse lg:flex-row gap-8 justify-between items-center">
		<h1 class="w-full">
			<Typewriter
				prefix={{
					word: "Wanna go ", className: "bg-gradient-to-b from-neutral-200 to-neutral-500 bg-clip-text font-bold text-transparent"
				}}
				texts={[
					{words: [
						{ word: "Blazingly Fast?", className: "bg-gradient-to-b text-yellow-600 to-orange-800 bg-clip-text font-bold text-transparent" },
					]},
					{words: [
						{ word: "High Available?", className: "bg-gradient-to-b from-red-600 to-red-900 bg-clip-text font-bold text-transparent" },
					]},
					{words: [
						{ word: "Zero Config?", className: "bg-gradient-to-b from-green-600 to-green-900 bg-clip-text font-bold text-transparent" },
					]},
					{words: [
						{ word: "Globally Cached?", className: "bg-gradient-to-b from-violet-800 to-violet-950 bg-clip-text font-bold text-transparent" },
					]},
				]}
				duration={1.5}
				className="w-full"
				cursorClassName="bg-gradient-to-b from-neutral-200 to-neutral-500"
			/>
		</h1>
		<Battleship className="w-full"></Battleship>
	</div>

</GridBackground>

<section>

</section>

<style>
</style>
