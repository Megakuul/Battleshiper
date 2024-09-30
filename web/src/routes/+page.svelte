<script>
	import EdgeIcon from "$lib/assets/EdgeIcon.svg"
	import WorkflowIcon from "$lib/assets/WorkflowIcon.svg"
  import Battleship from "$lib/components/Battleship.svelte";
  import Beam from "$lib/components/Beam.svelte";
	import GridBackground from "$lib/components/GridBackground.svelte";
  import Spotlight from "$lib/components/Spotlight.svelte";
  import SvelteIcon from "$lib/components/SvelteIcon.svelte";
  import Typewriter from "$lib/components/Typewriter.svelte";
  import PinContainer from "$lib/components/ui/pin/PinContainer.svelte";

	import {
		PUBLIC_SEO_DOMAIN
	} from "$env/static/public";
    import LogIcon from "$lib/components/LogIcon.svelte";
</script>

<svelte:head>
	<title>Home | Battleshiper</title>
	<meta name="description" content="Deploy your SvelteKit applications on automated, scalable infrastructure with no configuration worries." />
	<meta property="og:description" content="Deploy your SvelteKit applications on automated, scalable infrastructure with no configuration worries." />
	<meta property="og:title" content="Home - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}" />
</svelte:head>

<GridBackground gridBlockSize={64} gridColor="rgba(255, 255, 255, 0.2)" className="flex justify-center">
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
	<div class="10/12 sm:w-8/12 flex flex-col gap-16 my-24 sm:my-32">
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
				className="w-full flex justify-center"
				cursorClassName="bg-gradient-to-b from-neutral-200 to-neutral-500"
			/>
		</h1>
		<Battleship className="w-full "></Battleship>
	</div>
</GridBackground>

<Beam gradientStart="black" gradientVia="#757F9A" gradientEnd="#D7DDE8" className="flow-root z-0">
	<div class="flex flex-col lg:flex-row justify-evenly items-center my-64">
		<h1 class="text-xl lg:text-4xl text-center text-nowrap font-bold bg-slate-900/90 sm:bg-slate-700/20 rounded-lg p-4">Deploy SvelteKit in Seconds</h1>
		<PinContainer title="/kit.svelte.dev" href="https://kit.svelte.dev" containerClassName="my-16">
			<div class="flex h-[20rem] w-[18rem] sm:w-[20rem] basis-full flex-col p-4 text-slate-100/50 sm:basis-1/2">
				<h2 class="max-w-xs text-base font-bold text-slate-100">SvelteKit</h2>
				<div class="text-base font-normal text-slate-500">
					Battleshiper leverages Serverless Functions to power the GOAT framework.
				</div>
				<div class="mt-4 flex w-full h-full justify-center items-center bg-slate-900 brightness-75 rounded-lg">
					<SvelteIcon class="w-2/3 h-2/3" />
				</div>
			</div>
		</PinContainer>
	</div>

	<div class="flex flex-col-reverse lg:flex-row justify-evenly items-center my-64">
		<PinContainer title="/project" href="/project" containerClassName="my-16">
			<div class="flex h-[20rem] w-[18rem] sm:w-[20rem] basis-full flex-col p-4 text-slate-100/50 sm:basis-1/2">
				<h2 class="max-w-xs text-base font-bold text-slate-100">Edge Caching</h2>
				<div class="text-base font-normal text-slate-500">
					Assets and lazy-loaded chunks are cached directly at the Edge.
				</div>
				<div class="mt-4 flex w-full h-full justify-center items-center bg-slate-900 brightness-75 rounded-lg">
					<img alt="Edge Computing Icon" src={EdgeIcon} class="w-2/3 h-2/3" />
				</div>
			</div>
		</PinContainer>
		<h1 class="text-xl lg:text-4xl text-nowrap text-center font-bold bg-slate-900/90 sm:bg-slate-700/20 rounded-lg p-4">Profit from Global Caching</h1>
	</div>

	<div class="flex flex-col lg:flex-row justify-evenly items-center my-64">
		<h1 class="text-xl lg:text-4xl text-center text-nowrap font-bold bg-slate-900/90 sm:bg-slate-700/20 rounded-lg p-4">CI/CD? Relax.</h1>
		<PinContainer title="/project" href="/project" containerClassName="my-16">
			<div class="flex h-[20rem] w-[18rem] sm:w-[20rem] basis-full flex-col p-4 text-slate-100/50 sm:basis-1/2">
				<h2 class="max-w-xs text-base font-bold text-slate-100">Automated Deployment</h2>
				<div class="text-base font-normal text-slate-500">
					Your Application is automatically built and deployed to Battleshiper.
				</div>
				<div class="mt-4 flex w-full h-full justify-center items-center bg-slate-900 brightness-75 rounded-lg">
					<img alt="Workflow Icon" src={WorkflowIcon} class="w-2/3 h-2/3" />
				</div>
			</div>
		</PinContainer>
	</div>

	<div class="flex flex-col lg:flex-row justify-evenly items-center my-64">
		<PinContainer title="/logs" href="/logs" containerClassName="my-16">
			<div class="flex h-[20rem] w-[18rem] sm:w-[20rem] basis-full flex-col p-4 text-slate-100/50 sm:basis-1/2">
				<h2 class="max-w-xs text-base font-bold text-slate-100">Managed Logging</h2>
				<div class="text-base font-normal text-slate-500">
					Application Logs are automatically collected and stored in the Cloud.
				</div>
				<div class="mt-4 flex w-full h-full justify-center items-center bg-slate-900 brightness-75 rounded-lg">
					<LogIcon class="w-2/3 h-2/3"></LogIcon>
				</div>
			</div>
		</PinContainer>
		<h1 class="text-xl lg:text-4xl text-center text-nowrap font-bold bg-slate-900/90 sm:bg-slate-700/20 rounded-lg p-4">Effortless Logging</h1>
	</div>
</Beam>

<style>
</style>
