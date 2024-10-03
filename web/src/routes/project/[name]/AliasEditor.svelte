<script>
  import { Button } from "$lib/components/ui/button";
  import Icon from "@iconify/svelte";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import { fade } from "svelte/transition";
  import { flip } from 'svelte/animate';
  import { toast } from "svelte-sonner";
  import { Input } from "$lib/components/ui/input";
  import { UpdateAlias } from "$lib/adapter/resource/updatealias";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput}*/
  export let CurrentProjectRef;

  /** @type {string} */
  export let ExceptionRef;

  /** @type {string} */
  let currentAlias = "";

  /** @type {boolean} */
  let updateAliasButtonState;
</script>

<div class="flex flex-col gap-8 w-10/12 my-8">
  <div transition:fade class="flex flex-col items-start gap-4 w-full p-6 rounded-lg overflow-hidden bg-slate-700/20">
    <h1 class="text-xl md:text-2xl font-bold">Aliases</h1>
    <div class="flex flex-row w-full">
      <Input bind:value={currentAlias} type="text" placeholder="Alias" class="border-r-0 rounded-r-none mr-auto" />
      <Input disabled value=".{CurrentProjectRef?.name}" type="text" class="border-l-0 rounded-l-none w-36" />
    </div>
    <div class="flex flex-col gap-2 w-full h-[20vh] overflow-scroll-hidden">
      {#each Object.entries(CurrentProjectRef?.aliases ?? {}) as alias (alias[0])}
        <div class="flex flex-row w-full" animate:flip={{delay: 250, duration: 250}} transition:fade>
          <Input disabled value="{alias[0]}" type="text" class="w-full" />
          <Button variant="ghost" on:click={() => {
            if (CurrentProjectRef) {
              delete CurrentProjectRef.aliases[alias[0]];
              CurrentProjectRef.aliases = CurrentProjectRef.aliases;
            }
          }}>
            <Icon icon="line-md:minus-circle" class="" />
          </Button>
        </div>
      {/each}
    </div>
    <div class="flex flex-row gap-2 justify-start w-full">
      <Button type="submit" on:click={() => {
        if (CurrentProjectRef) {
          if (currentAlias === "") {
            // Alias apex
            CurrentProjectRef.aliases[CurrentProjectRef.name] = null;
          } else {
            CurrentProjectRef.aliases[`${currentAlias}.${CurrentProjectRef.name}`] = null;
          }
          currentAlias = "";
        }
      }}>
        Add <Icon icon="line-md:file-document-plus" class="ml-1" />
      </Button>

      <Button type="submit" class="ml-auto" on:click={async () => {
        try {
          if (!CurrentProjectRef) throw new Error("project not loaded");
          updateAliasButtonState = true;
          const projectOutput = await UpdateAlias({
            project_name: CurrentProjectRef.name,
            aliases: CurrentProjectRef.aliases,
          });
          toast.success("Success", {
            description: projectOutput.message
          })
          updateAliasButtonState = false;
          ExceptionRef = "";
        } catch (/** @type {any} */ err) {
          ExceptionRef = err.message;
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