<script>
  import { ListProject } from "$lib/adapter/resource/listproject";
  import * as Alert from "$lib/components/ui/alert";
  import { ProjectInfo } from "$lib/stores";
  import { CircleAlert } from "lucide-svelte";
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import Button from "$lib/components/ui/button/button.svelte";
  import ProjectQueryBar from "./ProjectQueryBar.svelte";
  import AdminQueryBar from "./AdminQueryBar.svelte";
  import LogPanel from "./LogPanel.svelte";
  import { PUBLIC_SEO_DOMAIN } from "$env/static/public";

  /**
   * @typedef {"project" | "admin"} LOG_MODE
  */

  /** @type {LOG_MODE} */
  let CurrentMode = "project";

  /** @type {import("$lib/adapter/resource/fetchlog").fetchLogInput}*/
  let CurrentProjectLogInput = {
    project_name: "",
    log_type: "",
    count: 10,
    start_time: new Date(Date.now() - 1000 * 60 * 60).getTime(), // 1 hour before now
    end_time: Date.now(), // Now
  };

  /** @type {import("$lib/adapter/resource/fetchlog").fetchLogOutput|undefined}*/
  let CurrentProjectLogOutput;

  /** @type {import("$lib/adapter/admin/fetchlog").fetchLogInput}*/
  let CurrentAdminLogInput = {
    log_type: "",
    count: 10,
    start_time: new Date(Date.now() - 1000 * 60 * 60).getTime(), // 1 hour before now
    end_time: Date.now(), // Now
  };

  /** @type {import("$lib/adapter/admin/fetchlog").fetchLogOutput|undefined}*/
  let CurrentAdminLogOutput;

  /** @type {string} */
  let Exception = "";

  onMount(async () => {
    try {
      if (!$ProjectInfo) {
        $ProjectInfo = await ListProject();
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
	<title>Logs | Battleshiper</title>
	<meta name="description" content="Query and analyze the logs of your system." />
	<meta property="og:description" content="Query and analyze the logs of your system." />
	<meta property="og:title" content="Logs - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}/logs" />
</svelte:head>


{#if CurrentMode==="admin"}
<div class="flex flex-col items-center w-full gap-4 my-10">
  <LogPanel LogEvents={CurrentAdminLogOutput?.events ?? []} />

  <AdminQueryBar bind:CurrentLogInputRef={CurrentAdminLogInput} bind:CurrentLogOutputRef={CurrentAdminLogOutput} bind:ExceptionRef={Exception} />

  <Button variant="ghost" class="w-10/12" on:click={() => CurrentMode = "project"}>Switch to project mode</Button>
</div>
{:else if CurrentMode==="project"}
<div class="flex flex-col items-center w-full gap-4 my-10">
  <LogPanel LogEvents={CurrentProjectLogOutput?.events ?? []} />

  <ProjectQueryBar bind:CurrentLogInputRef={CurrentProjectLogInput} bind:CurrentLogOutputRef={CurrentProjectLogOutput} bind:ExceptionRef={Exception} />

  <Button variant="ghost" class="w-10/12" on:click={() => CurrentMode = "admin"}>Switch to admin mode</Button>
</div>
{/if}


{#if Exception}
  <div transition:fade class="flex flex-col items-center w-full mb-10">
    <Alert.Root variant="destructive" class="w-10/12">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Exception}</Alert.Description>
    </Alert.Root>
  </div>
{/if}