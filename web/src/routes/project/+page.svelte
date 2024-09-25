<script>
  import { ListProject } from "$lib/adapter/resource/listproject";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button } from "$lib/components/ui/button";
  import * as Avatar from "$lib/components/ui/avatar";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import Icon from '@iconify/svelte';
  import { onMount } from "svelte";
  import { RegisterUser } from "$lib/adapter/user/registeruser";
  import { Authorize } from "$lib/adapter/auth/authorize";
  import { fade } from "svelte/transition";
  import BorderBeam from "$lib/components/BorderBeam.svelte";
  import { goto } from "$app/navigation";
  import { ProjectInfo } from "$lib/stores";
  import { toast } from "svelte-sonner";
  import { 
    PUBLIC_SEO_DOMAIN 
  } from "$env/static/public";

  /** @type {string} */
  let Error = "";

  let Hostname = "";

  onMount(async () => {
    Hostname = window.location.hostname;
    try {
      if (!$ProjectInfo) {
        $ProjectInfo = await ListProject();
      }
    } catch (/** @type {any} */ err) {
      Error = err.message;
    }
  })
</script>

<svelte:head>
	<title>Projects | Battleshiper</title>
	<meta name="description" content="Manage all Battleshiper projects in one place." />
	<meta property="og:description" content="Manage all Battleshiper projects in one place." />
	<meta property="og:title" content="Projects - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}/project" />
</svelte:head>

{#if $ProjectInfo}
  <div class="flex flex-col gap-8 items-center mt-12 mb-16">
    <h1 class="text-6xl font-bold text-center text-slate-200/80 ">Projects</h1>
    <div class="flex flex-row gap-2">
      <Button variant="ghost" class="text-sm md:text-xl" on:click={async () => {
        try {
          $ProjectInfo = await ListProject();
          toast("Success", {
            description: "Projects refreshed",
          })
        } catch (/** @type {any} */ err) {
          Error = err.message;
          toast("Error", {
            description: err.message,
          })
        }
      }}>Refresh Projects <Icon icon="line-md:rotate-270" class="ml-1" />
      </Button>
      <Button variant="ghost" class="text-sm md:text-xl" on:click={() => goto("/project/new")}>
        Create new Project <Icon icon="line-md:folder-plus" class="ml-1" />
      </Button>
    </div>
  </div>
  <div class="flex flex-col gap-12 mb-10 items-center h-[80vh] overflow-scroll-hidden">
    {#each $ProjectInfo.projects as project}
      <a 
        href="/project/{project.name}" 
        class="relative w-9/12 group flex flex-row items-center p-6 rounded-lg overflow-hidden cursor-pointer bg-slate-900/10 border-[1px] border-slate-200/15">
        <BorderBeam size={150} duration={10} colorFrom="#304352" colorTo="#d7d2cc" class="transition-all duration-700 opacity-0 group-hover:opacity-100" />
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Avatar.Root class="h-16 md:h-20 w-16 md:w-20 m-1 md:m-0 p-0 md:p-1">
              <Avatar.Image 
                class="{project.status ? "border-4 border-red-800" : ""}" 
                src="https://{project.name}.{Hostname}/favicon.png" 
                alt="{project.name} favicon" >
              </Avatar.Image>
              <Avatar.Fallback class="w-16 h-16 text-2xl font-bold uppercase {project.status ? "border-4 border-red-800" : ""}">
                {project.name[0] ?? "-"}
              </Avatar.Fallback>
            </Avatar.Root>
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p class="text-orange-700">{project.status}</p>
          </Tooltip.Content>
        </Tooltip.Root>
        <div class="flex flex-col ml-1">
          <h1 class="text-2xl font-bold">{project.name}</h1>
          <a class="text-sm sm:text-lg underline text-slate-300" href="https://{project.name}.{Hostname}">
            {project.name}.{Hostname}<Icon icon="line-md:external-link" class="hidden sm:inline ml-1" />
          </a>
        </div>
        <Tooltip.Root>
          <Tooltip.Trigger class="ml-auto">
            {#if project.deleted}
            <div class="w-4 h-4 rounded-full bg-red-900"></div>
            {:else if !project.initialized}
            <div class="w-4 h-4 rounded-full bg-blue-600"></div>
            {:else}
            <div class="w-4 h-4 rounded-full bg-green-800"></div>
            {/if}
          </Tooltip.Trigger>
          <Tooltip.Content>
            {#if project.deleted}
            <p>Project is beeing deleted...</p>
            {:else if !project.initialized}
            <p>Project is beeing initialized...</p>
            {:else}
            <p>Project is running...</p>
            {/if}
          </Tooltip.Content>
        </Tooltip.Root>
      </a>
    {/each}
  </div>
{:else if Error}
  <div transition:fade class="min-h-[60vh] flex justify-center items-center">
    <h1 class="text-3xl md:text-6xl text-center opacity-80">Oops... projects hit a snag!</h1>
  </div>
  <div transition:fade class="m-6 flex flex-col space-y-4">
    <Tooltip.Root>
      <Tooltip.Trigger asChild let:builder>
        <Button builders={[builder]} variant="outline" class="text-2xl h-20" on:click={Authorize}>
          LOG IN <Icon icon="line-md:github-loop" class="ml-2" />
        </Button>
      </Tooltip.Trigger>
      <Tooltip.Content>
        <p>Log in with Github</p>
      </Tooltip.Content>
    </Tooltip.Root>
    <Alert.Root variant="destructive">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Error}</Alert.Description>
    </Alert.Root>
  </div>
{:else}
  <div transition:fade class="min-h-[80vh] flex justify-center items-center">
    <LoaderCircle class="mr-2 h-16 w-16 opacity-80 animate-spin" />
  </div>
{/if}