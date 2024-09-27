<script>
  import { Button } from "$lib/components/ui/button";
  import Icon from "@iconify/svelte";
  import { fade } from "svelte/transition";
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";

  /** @type {URLSearchParams|undefined}*/
  let queryParams; 

  onMount(() => {
    queryParams = new URLSearchParams(window.location.search);
  })
</script>

<div transition:fade class="min-h-[60vh] flex flex-col gap-4 justify-center items-center">
  <div class="flex flex-row items-center gap-4">
    <Icon icon="line-md:cloud-alt-braces-loop" class="text-3xl md:text-6xl opacity-80"/>
    <h1 class="text-3xl md:text-6xl text-center font-bold opacity-80">
      {#if queryParams && queryParams.get("type")}
        {queryParams.get("type")}
      {:else}
        Unexpected Error
      {/if}
    </h1>
  </div>
  <h2 class="text-xs md:text-base text-center w-3/4 opacity-70">
    {#if queryParams && queryParams.get("message")}
      {queryParams.get("message")}
    {:else}
      The Battleshiper system encountered an unexpected issue
    {/if}
  </h2>
  
</div>
<div transition:fade class="m-6 flex flex-col space-y-4">
  <Button variant="outline" class="text-2xl h-20" on:click={() => goto("/")}>
    Return to Homepage <Icon icon="line-md:rotate-180" class="ml-2" />
  </Button>
</div>
