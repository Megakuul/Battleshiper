<script>
  import { page } from "$app/stores";
  import { ListProject } from "$lib/adapter/resource/listproject";
  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import { ProjectInfo } from "$lib/stores";
  import Icon from "@iconify/svelte";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import { goto } from "$app/navigation";
  import StatusBar from "./StatusBar.svelte";
  import AliasEditor from "./AliasEditor.svelte";
  import ConfigEditor from "./ConfigEditor.svelte";
  import PipelineState from "./PipelineState.svelte";
    import ActionBar from "./ActionBar.svelte";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput|undefined}*/
  let CurrentProject;

  /** @type {string}*/
  let Exception = "";

  /** @type {string} */
  let Hostname = "";

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

    <StatusBar bind:CurrentProjectRef={CurrentProject} bind:ExceptionRef={Exception} Hostname={Hostname} />

    <PipelineState bind:CurrentProjectRef={CurrentProject} />

    <AliasEditor bind:CurrentProjectRef={CurrentProject} bind:ExceptionRef={Exception} />

    <ConfigEditor bind:CurrentProjectRef={CurrentProject} bind:ExceptionRef={Exception} />

    <ActionBar bind:CurrentProjectRef={CurrentProject} bind:ExceptionRef={Exception} />

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