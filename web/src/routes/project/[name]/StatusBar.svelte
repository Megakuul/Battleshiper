<script>
  import { page } from "$app/stores";
  import { ListProject } from "$lib/adapter/resource/listproject";
  import * as Tooltip from "$lib/components/ui/tooltip";
  import { Button } from "$lib/components/ui/button";
  import { ProjectInfo } from "$lib/stores";
  import Icon from "@iconify/svelte";
  import { toast } from "svelte-sonner";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput|undefined}*/
  export let CurrentProjectRef;

  /** @type {string} */
  export let ExceptionRef;

  /** @type {string} */
  export let Hostname;
</script>

<div class="flex flex-col md:flex-row gap-8 w-10/12">
  <div class="flex flex-col items-start gap-4 w-full md:w-1/3 p-6 rounded-lg max-h-[25vh] overflow-hidden bg-slate-700/20">
    <h1 class="text-xl md:text-2xl font-bold">{CurrentProjectRef?.name}</h1>
    <a class="text-xs sm:text-sm font-bold underline text-slate-300" href="https://{CurrentProjectRef?.name}.{Hostname}" target={'_blank'} rel="noopener noreferrer">
      Project Endpoint<Icon icon="line-md:external-link" class="hidden sm:inline ml-1" />
    </a>
    <a class="text-xs sm:text-sm font-bold underline text-slate-300" href="{CurrentProjectRef?.repository.url}" target={'_blank'} rel="noopener noreferrer">
      Source Repository<Icon icon="mdi:source-repository" class="hidden sm:inline ml-1" />
    </a>
    <Button class="text-sm md:text-xl mt-auto w-full" on:click={async () => {
      try {
        $ProjectInfo = await ListProject();
        CurrentProjectRef = $ProjectInfo.projects.find(project => project.name === $page.params.name);
        if (!CurrentProjectRef) {
          ExceptionRef = "Project not found"
        }
        toast("Success", {
          description: "Projects refreshed",
        })
        ExceptionRef = "";
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
        toast("Exception", {
          description: err.message,
        })
      }
    }}>Refresh <Icon icon="line-md:rotate-270" class="ml-1" />
    </Button>
  </div>

  <div class="relative flex flex-col gap-4 w-full p-6 rounded-lg max-h-[25vh] overflow-scroll-hidden bg-slate-700/20">
    <h1 class="text-xl md:text-2xl font-bold">
      Status: 
      {#if !CurrentProjectRef?.status}
        <span class="text-green-700 uppercase block md:inline">System operational</span>
      {:else}
        <span class="text-red-700 uppercase block md:inline">{CurrentProjectRef.status.slice(0, CurrentProjectRef.status.indexOf(":"))}</span>
      {/if}
    </h1>
    <p class="text-xs md:text-xl text-red-700 p-2 rounded-lg break-all bg-slate-600/20">
      {CurrentProjectRef?.status.slice(CurrentProjectRef.status.indexOf(":") + 1)}
    </p>
    <Tooltip.Root>
      <Tooltip.Trigger class="absolute top-6 right-6">
        {#if CurrentProjectRef?.deleted}
        <div class="w-6 h-6 rounded-full bg-red-900"></div>
        {:else if !CurrentProjectRef?.initialized}
        <div class="w-6 h-6 rounded-full bg-blue-600"></div>
        {:else}
        <div class="w-6 h-6 rounded-full bg-green-800"></div>
        {/if}
      </Tooltip.Trigger>
      <Tooltip.Content>
        {#if CurrentProjectRef?.deleted}
        <p>Project is beeing deleted...</p>
        {:else if !CurrentProjectRef?.initialized}
        <p>Project is beeing initialized...</p>
        {:else}
        <p>Project is running...</p>
        {/if}
      </Tooltip.Content>
    </Tooltip.Root>
  </div>
</div>