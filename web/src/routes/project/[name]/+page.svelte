<script>
  import { page } from "$app/stores";
  import { ListProject } from "$lib/adapter/resource/listproject";
  import * as Tooltip from "$lib/components/ui/tooltip";
  import * as Alert from "$lib/components/ui/alert";
  import * as Select from "$lib/components/ui/select";
  import { Button } from "$lib/components/ui/button";
  import { ProjectInfo } from "$lib/stores";
  import Icon from "@iconify/svelte";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import { goto } from "$app/navigation";
  import { flip } from 'svelte/animate';
  import { toast } from "svelte-sonner";
  import { Input } from "$lib/components/ui/input";
  import { UpdateProject } from "$lib/adapter/resource/updateproject";
  import { UpdateAlias } from "$lib/adapter/resource/updatealias";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput|undefined}*/
  let CurrentProject;

  /** @type {string} */
  let CurrentAlias = "";

  /** @type {string}*/
  let Exception = "";

  /** @type {string} */
  let Hostname = "";

  /** @type {boolean} */
  let updateAliasButtonState;

  /** @type {boolean} */
  let updateButtonState;

  onMount(async () => {
    try {
      Hostname = window.location.hostname;
      if (!$ProjectInfo) {
        $ProjectInfo = await ListProject();
      }
      CurrentProject = $ProjectInfo.projects.find(project => project.name === $page.params.name);
      if (!CurrentProject) {
        Exception = "Project not found"
      }
    } catch (/** @type {any} */ err) {
      Exception = err.message;
    }
  })

  // Generate a user friendly message if the exception is longer then 250 chars.
  // This is primarely for unexpected errors that cause the api to return an error page in html format.
  $: if (Exception && Exception.length > 250) Exception = "Unexpected error occured";
</script>

<svelte:head>
	<title>Project | Battleshiper</title>
  <meta name="robots" content="noindex, follow">
</svelte:head>

{#if CurrentProject}
  <div transition:fade class="min-h-[60vh] flex flex-col items-center mt-12 mb-16">
    <h1 class="text-6xl font-bold text-center text-slate-200/80 mb-20">Project Configuration</h1>

    <div class="flex flex-col md:flex-row gap-8 w-10/12">
      <div class="flex flex-col items-start gap-4 w-full md:w-1/3 p-6 rounded-lg max-h-[25vh] overflow-hidden bg-slate-700/20">
        <h1 class="text-xl md:text-2xl font-bold">{CurrentProject.name}</h1>
        <a class="text-xs sm:text-sm font-bold underline text-slate-300" href="https://{CurrentProject.name}.{Hostname}">
          Project Endpoint<Icon icon="line-md:external-link" class="hidden sm:inline ml-1" />
        </a>
        <a class="text-xs sm:text-sm font-bold underline text-slate-300" href="{CurrentProject.repository.url}">
          Source Repository<Icon icon="mdi:source-repository" class="hidden sm:inline ml-1" />
        </a>
        <Button class="text-sm md:text-xl mt-auto w-full" on:click={async () => {
          try {
            $ProjectInfo = await ListProject();
            CurrentProject = $ProjectInfo.projects.find(project => project.name === $page.params.name);
            if (!CurrentProject) {
              Exception = "Project not found"
            }
            toast("Success", {
              description: "Projects refreshed",
            })
          } catch (/** @type {any} */ err) {
            Exception = err.message;
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
          {#if !CurrentProject.status}
            <span class="text-green-700 uppercase block md:inline">System operational</span>
          {:else}
            <span class="text-red-700 uppercase block md:inline">{CurrentProject.status.slice(0, CurrentProject.status.indexOf(":"))}</span>
          {/if}
        </h1>
        <p class="text-xs md:text-xl text-red-700 p-2 rounded-lg break-all bg-slate-600/20">
          {CurrentProject.status.slice(CurrentProject.status.indexOf(":") + 1)}
        </p>
        <Tooltip.Root>
          <Tooltip.Trigger class="absolute top-6 right-6">
            {#if CurrentProject.deleted}
            <div class="w-6 h-6 rounded-full bg-red-900"></div>
            {:else if !CurrentProject.initialized}
            <div class="w-6 h-6 rounded-full bg-blue-600"></div>
            {:else}
            <div class="w-6 h-6 rounded-full bg-green-800"></div>
            {/if}
          </Tooltip.Trigger>
          <Tooltip.Content>
            {#if CurrentProject.deleted}
            <p>Project is beeing deleted...</p>
            {:else if !CurrentProject.initialized}
            <p>Project is beeing initialized...</p>
            {:else}
            <p>Project is running...</p>
            {/if}
          </Tooltip.Content>
        </Tooltip.Root>
      </div>
    </div>

    <div class="flex flex-col gap-8 w-10/12 my-8">
      <div class="flex flex-col items-start gap-4 w-full p-6 rounded-lg max-h-[25vh] overflow-hidden bg-slate-700/20">
        <h1 class="text-xl md:text-2xl font-bold">{CurrentProject.name}</h1>

      </div>
    </div>

    <div class="flex flex-col gap-8 w-10/12 my-8">
      <div transition:fade class="flex flex-col items-start gap-4 w-full p-6 rounded-lg overflow-hidden bg-slate-700/20">
        <h1 class="text-xl md:text-2xl font-bold">Aliases</h1>
        <div class="flex flex-row w-full">
          <Input bind:value={CurrentAlias} type="text" placeholder="Alias" class="border-r-0 rounded-r-none mr-auto" />
          <Input disabled value=".{CurrentProject.name}" type="text" class="border-l-0 rounded-l-none w-max" />
        </div>
        <div class="flex flex-col gap-2 w-full h-[20vh] overflow-scroll-hidden">
          {#each Object.entries(CurrentProject.aliases) as alias, i (alias[0])}
            <div class="flex flex-row w-full" animate:flip={{delay: 250, duration: 250}} transition:fade>
              <Input disabled value="{alias[0]}" type="text" class="w-full" />
              <Button variant="ghost" on:click={() => {
                if (CurrentProject) {
                  delete CurrentProject.aliases[alias[0]];
                  CurrentProject.aliases = CurrentProject.aliases;
                }
              }}>
                <Icon icon="line-md:minus-circle" class="" />
              </Button>
            </div>
          {/each}
        </div>
        <div class="flex flex-row gap-2 justify-start w-full">
          <Button type="submit" on:click={() => {
            if (CurrentProject) {
              if (CurrentAlias === "") {
                // Alias apex
                CurrentProject.aliases[CurrentProject.name] = null;
              } else {
                CurrentProject.aliases[`${CurrentAlias}.${CurrentProject.name}`] = null;
              }
              CurrentAlias = "";
            }
          }}>
            Add <Icon icon="line-md:file-document-plus" class="ml-1" />
          </Button>

          <Button type="submit" class="ml-auto" on:click={async () => {
            try {
              if (!CurrentProject) throw new Error("project not loaded");
              updateAliasButtonState = true;
              const projectOutput = await UpdateAlias({
                project_name: CurrentProject.name,
                aliases: CurrentProject.aliases,
              });
              toast.success("Success", {
                description: projectOutput.message
              })
              updateAliasButtonState = false;
            } catch (/** @type {any} */ err) {
              Exception = err.message;
              toast.error("Error", {
                description: "Failed deploy aliases",
              })
            }
            updateAliasButtonState = false;
          }}>
            Deploy Aliases
            {#if updateAliasButtonState}
              <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
            {/if}
          </Button>
        </div>
      </div>
    </div>

    <div class="flex flex-col gap-8 w-10/12 my-8">
      <div class="flex flex-col items-start gap-4 w-full p-6 rounded-lg overflow-hidden bg-slate-700/20">
        <h1 class="text-xl md:text-2xl font-bold">Configuration</h1>
        <div class="flex flex-row gap-4 w-full">
          <Input disabled bind:value={CurrentProject.repository.url} type="text" placeholder="Repository" class="" />
          <Input bind:value={CurrentProject.repository.branch} type="text" placeholder="Branch" class="w-[100px]" />
        </div>
        <Input disabled bind:value={CurrentProject.build_image} type="text" placeholder="Build Image" />
        <Input bind:value={CurrentProject.build_command} type="text" placeholder="Build Command" />
        <Input bind:value={CurrentProject.output_directory} type="text" placeholder="Build Output Directory" />
        <Button class="w-full" type="submit" on:click={async () => {
          try {
            if (!CurrentProject) throw new Error("project not loaded");
            updateButtonState = true;
            const projectOutput = await UpdateProject({
              project_name: CurrentProject.name,
              build_command: CurrentProject.build_command,
              output_directory: CurrentProject.output_directory,
              repository: {
                id: CurrentProject.repository.id,
                url: CurrentProject.repository.url,
                branch: CurrentProject.repository.branch,
              }
            });
            toast.success("Success", {
              description: projectOutput.message
            })
            updateButtonState = false;
          } catch (/** @type {any} */ err) {
            Exception = err.message;
            toast.error("Error", {
              description: "Failed to update project",
            })
          }
          updateButtonState = false;
        }}>
          Update Project
          {#if updateButtonState}
            <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
          {/if}
        </Button>
      </div>
    </div>

    {#if Exception}
    <div transition:fade class="flex flex-col items-center w-full">
      <Alert.Root variant="destructive" class="w-10/12">
        <CircleAlert class="h-4 w-4" />
        <Alert.Title>Error</Alert.Title>
        <Alert.Description>{Exception}</Alert.Description>
      </Alert.Root>
    </div>
    {/if}
  </div>
{:else if Exception}
  <div transition:fade class="min-h-[60vh] flex flex-row gap-4 justify-center items-center">
    <Icon icon="line-md:compass-off-loop" class="text-3xl md:text-6xl opacity-80"/>
    <h1 class="text-3xl md:text-6xl text-center opacity-80">Project Not Found</h1>
  </div>
  <div transition:fade class="m-6 flex flex-col space-y-4">
    <Button variant="outline" class="text-2xl h-20" on:click={() => goto("/project")}>
      Return to Projects <Icon icon="line-md:rotate-180" class="ml-2" />
    </Button>
    <Alert.Root variant="destructive">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Exception}</Alert.Description>
    </Alert.Root>
  </div>
{:else}
  <div transition:fade class="min-h-[80vh] flex justify-center items-center">
    <LoaderCircle class="mr-2 h-16 w-16 opacity-80 animate-spin" />
  </div>
{/if}