<script>
	import "../app.css";
	import BattleshiperIcon from "$lib/assets/BattleshiperIcon.svg"
  import * as Sheet from "$lib/components/ui/sheet";
  import { Button } from "$lib/components/ui/button";
  import { Toaster } from "$lib/components/ui/sonner";

	import {
		PUBLIC_IMPRINT_HREF,
		PUBLIC_PRIVACY_HREF,
		PUBLIC_TOS_HREF,
		PUBLIC_CONTACT_1,
		PUBLIC_CONTACT_2,
		PUBLIC_CONTACT_3
	} from "$env/static/public";
    import { goto } from "$app/navigation";
</script>

<Toaster />

<div class="app">
	<nav class="h-24 flex flex-row w-full lg:w-auto justify-evenly items-center bg-black">
		<a class="flex flex-row gap-2 items-center" href="/">
			<img src="{BattleshiperIcon}" alt="Battleshiper Icon" class="w-10" />
			<h1 class="text-2xl sm:text-3xl font-bold">Battleshiper</h1>
		</a>

		<a 
			class="text-2xl font-medium hidden lg:block hover:bg-white hover:bg-opacity-5 px-8 py-3 rounded-xl transition-all duration-700" 
			href="/project">Projects
		</a>
		<a 
			class="text-2xl font-medium hidden lg:block hover:bg-white hover:bg-opacity-5 px-8 py-3 rounded-xl transition-all duration-700" 
			href="/logs">Logs
		</a>
		<a 
			class="text-2xl font-medium hidden lg:block hover:bg-white hover:bg-opacity-5 px-8 py-3 rounded-xl transition-all duration-700" 
			href="/admin">Admin
		</a>
		<a 
			class="text-2xl font-medium hidden lg:block hover:bg-white hover:bg-opacity-5 px-8 py-3 rounded-xl transition-all duration-700" 
			href="/profile">Profile
		</a>

		<Sheet.Root>
			<Sheet.Trigger asChild let:builder>
				<Button builders={[builder]} variant="ghost" class="block lg:hidden">
					<svg stroke="currentColor" fill="currentColor" stroke-width="0" viewBox="0 0 24 24" aria-hidden="true" class="text-2xl" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path fill-rule="evenodd" d="M3 6.75A.75.75 0 013.75 6h16.5a.75.75 0 010 1.5H3.75A.75.75 0 013 6.75zM3 12a.75.75 0 01.75-.75h16.5a.75.75 0 010 1.5H3.75A.75.75 0 013 12zm8.25 5.25a.75.75 0 01.75-.75h8.25a.75.75 0 010 1.5H12a.75.75 0 01-.75-.75z" clip-rule="evenodd"></path></svg>
				</Button>
			</Sheet.Trigger>
			<Sheet.Content side="right" class="bg-black border-transparent flex flex-col gap-4">
				<Sheet.Header>
					<Sheet.Title class="flex flex-row items-center gap-2">
						<h1 class="text-2xl">Battleshiper</h1>
						<img src="{BattleshiperIcon}" alt="Battleshiper Icon" class="w-8" />
					</Sheet.Title>
				</Sheet.Header>
					<Sheet.Close asChild let:builder>
						<Button builders={[builder]} 
							variant="ghost" 
							class="text-xl font-medium hover:bg-slate-600/20 justify-start duration-700"
							on:click={() => goto("/")}>Home
						</Button>
					</Sheet.Close>
					<Sheet.Close asChild let:builder>
						<Button builders={[builder]} 
							variant="ghost" 
							class="text-xl font-medium hover:bg-slate-600/20 justify-start duration-700"
							on:click={() => goto("/project")}>Projects
						</Button>
					</Sheet.Close>
					<Sheet.Close asChild let:builder>
						<Button builders={[builder]} 
							variant="ghost" 
							class="text-xl font-medium hover:bg-slate-600/20 justify-start duration-700"
							on:click={() => goto("/logs")}>Logs
						</Button>
					</Sheet.Close>
					<Sheet.Close asChild let:builder>
						<Button builders={[builder]} 
							variant="ghost" 
							class="text-xl font-medium hover:bg-slate-600/20 justify-start duration-700"
							on:click={() => goto("/admin")}>Admin
						</Button>
					</Sheet.Close>
					<Sheet.Close asChild let:builder>
						<Button builders={[builder]} 
							variant="ghost" 
							class="text-xl font-medium hover:bg-slate-600/20 justify-start duration-700"
							on:click={() => goto("/profile")}>Profile
						</Button>
					</Sheet.Close>
			</Sheet.Content>
		</Sheet.Root>
	</nav>

	<main class="overflow-hidden">
		<slot></slot>
	</main>

	<footer class="w-full flex flex-col sm:flex-row justify-start items-start p-6 bg-slate-950">
		<section class="w-full flex flex-col">
			<img alt="Battleshiper Icon" src={BattleshiperIcon} class="w-28 my-2 animate-pulse" />
			<h1 class="text-xl font-bold">Battleshiper</h1>
			<h2 class="text-sm">Deployment Platform</h2>
		</section>
		<div class="w-full flex flex-col sm:flex-row gap-8 justify-around mt-6">
			<section class="flex flex-col gap-2">
				<h1 class="text-slate-600 font-semibold opacity-70">Legal</h1>
				{#if PUBLIC_IMPRINT_HREF}
					<a class="text-slate-400 opacity-90 hover:underline" href="{PUBLIC_IMPRINT_HREF}">Imprint</a>
				{/if}
				{#if PUBLIC_PRIVACY_HREF}
					<a class="text-slate-400 opacity-90 hover:underline" href="{PUBLIC_PRIVACY_HREF}">Privacy Policy</a>
				{/if}
				{#if PUBLIC_TOS_HREF}
					<a class="text-slate-400 opacity-90 hover:underline" href="{PUBLIC_TOS_HREF}">Terms of Services</a>
				{/if}
			</section>
			<section class="flex flex-col gap-2">
				<h1 class="text-slate-600 font-semibold opacity-70">Contact</h1>
				{#if PUBLIC_CONTACT_1}
					<p class="text-slate-400 text-sm">{PUBLIC_CONTACT_1}</p>
				{/if}
				{#if PUBLIC_CONTACT_2}
					<p class="text-slate-400 text-sm">{PUBLIC_CONTACT_2}</p>
				{/if}
				{#if PUBLIC_CONTACT_3}
					<p class="text-slate-400 text-sm">{PUBLIC_CONTACT_3}</p>
				{/if}
			</section>
		</div>
	</footer>
</div>


<style>
</style>
