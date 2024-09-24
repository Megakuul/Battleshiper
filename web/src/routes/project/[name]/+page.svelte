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

  /** @type {import("$lib/adapter/resource/listproject").projectOutput|undefined}*/
  let CurrentProject;

  /** @type {string}*/
  let Error = "";

  onMount(async () => {
    try {
      if (!$ProjectInfo) {
        $ProjectInfo = await ListProject();
      }
      CurrentProject = $ProjectInfo.projects.find(project => project.name === $page.params.name);
      if (!CurrentProject) {
        Error = "Project not found"
      }
    } catch (/** @type {any} */ err) {
      Error = err.message;
    }
  })
</script>

{#if CurrentProject}
  <h1>Project {CurrentProject.name}</h1>
{:else if Error}
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
      <Alert.Description>{Error}</Alert.Description>
    </Alert.Root>
  </div>
{:else}
  <div transition:fade class="min-h-[80vh] flex justify-center items-center">
    <LoaderCircle class="mr-2 h-16 w-16 opacity-80 animate-spin" />
  </div>
{/if}
