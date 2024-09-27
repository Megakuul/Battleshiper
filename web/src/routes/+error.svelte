<script>
  import { Button } from "$lib/components/ui/button";
  import Icon from "@iconify/svelte";
  import { fade } from "svelte/transition";
  import { goto } from "$app/navigation";
</script>

<div transition:fade class="min-h-[60vh] flex flex-row gap-4 justify-center items-center">
  <Icon icon="line-md:compass-off-loop" class="text-3xl md:text-6xl opacity-80"/>
  <h1 class="text-3xl md:text-6xl text-center opacity-80">
    Page Not Found
  </h1>
  
</div>
<div transition:fade class="m-6 flex flex-col space-y-4">
  <Button variant="outline" class="text-2xl h-20" on:click={() => goto("/")}>
    Return to Homepage <Icon icon="line-md:rotate-180" class="ml-2" />
  </Button>
</div>
