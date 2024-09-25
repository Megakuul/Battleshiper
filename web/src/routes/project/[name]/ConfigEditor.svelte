
<script>
  import { Button } from "$lib/components/ui/button";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import { toast } from "svelte-sonner";
  import { Input } from "$lib/components/ui/input";
  import { UpdateProject } from "$lib/adapter/resource/updateproject";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput}*/
  export let CurrentProjectRef;

  /** @type {string} */
  export let ExceptionRef;

  /** @type {boolean} */
  let updateButtonState;
</script>

<div class="flex flex-col gap-8 w-10/12 my-8">
  <div class="flex flex-col items-start gap-4 w-full p-6 rounded-lg overflow-hidden bg-slate-700/20">
    <h1 class="text-xl md:text-2xl font-bold">Configuration</h1>
    <div class="flex flex-row gap-4 w-full">
      <Input disabled bind:value={CurrentProjectRef.repository.url} type="text" placeholder="Repository" class="" />
      <Input bind:value={CurrentProjectRef.repository.branch} type="text" placeholder="Branch" class="w-[100px]" />
    </div>
    <Input disabled bind:value={CurrentProjectRef.build_image} type="text" placeholder="Build Image" />
    <Input bind:value={CurrentProjectRef.build_command} type="text" placeholder="Build Command" />
    <Input bind:value={CurrentProjectRef.output_directory} type="text" placeholder="Build Output Directory" />
    <Button class="w-full" type="submit" on:click={async () => {
      try {
        if (!CurrentProjectRef) throw new Error("project not loaded");
        updateButtonState = true;
        const projectOutput = await UpdateProject({
          project_name: CurrentProjectRef.name,
          build_command: CurrentProjectRef.build_command,
          output_directory: CurrentProjectRef.output_directory,
          repository: {
            id: CurrentProjectRef.repository.id,
            url: CurrentProjectRef.repository.url,
            branch: CurrentProjectRef.repository.branch,
          }
        });
        toast.success("Success", {
          description: projectOutput.message
        })
        updateButtonState = false;
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
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