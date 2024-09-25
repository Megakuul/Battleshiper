
<script>
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import { toast } from "svelte-sonner";
  import { UpdateProject } from "$lib/adapter/resource/updateproject";
  import { BuildProject } from "$lib/adapter/resource/buildproject";
  import { DeleteProject } from "$lib/adapter/admin/deleteproject";
    import { cn } from "$lib/utils";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput}*/
  export let CurrentProjectRef;

  /** @type {string} */
  export let ExceptionRef;

  /** @type {boolean} */
  let buildButtonState;

  /** @type {boolean} */
  let deleteButtonState;
</script>

<div class="flex flex-row gap-8 w-10/12 my-8">
  <Dialog.Root>
    <Dialog.Trigger class={cn(buttonVariants({ variant: "default" }), "w-full")}>Build Project</Dialog.Trigger>
    <Dialog.Content>
      <Dialog.Header>
        <Dialog.Title>Manually initiate project build?</Dialog.Title>
        <Dialog.Description>
          <Button class="w-full mt-6" type="submit" on:click={async () => {
            try {
              if (!CurrentProjectRef) throw new Error("project not loaded");
              buildButtonState = true;
              const projectOutput = await BuildProject({
                project_name: CurrentProjectRef.name,
              });
              toast.success("Success", {
                description: projectOutput.message
              })
              buildButtonState = false;
            } catch (/** @type {any} */ err) {
              ExceptionRef = err.message;
              toast.error("Error", {
                description: "Failed to initiate project build",
              })
            }
            buildButtonState = false;
          }}>
            Build Project
            {#if buildButtonState}
              <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
            {/if}
          </Button>
        </Dialog.Description>
      </Dialog.Header>
    </Dialog.Content>
  </Dialog.Root>

  <Dialog.Root>
    <Dialog.Trigger class={cn(buttonVariants({ variant: "destructive" }), "w-full")}>Delete Project</Dialog.Trigger>
    <Dialog.Content>
      <Dialog.Header>
        <Dialog.Title>Delete project and associated data?</Dialog.Title>
        <Dialog.Description>
          <Button variant="destructive" class="w-full mt-6" type="submit" on:click={async () => {
            try {
              if (!CurrentProjectRef) throw new Error("project not loaded");
              deleteButtonState = true;
              const projectOutput = await DeleteProject({
                project_name: CurrentProjectRef.name,
              });
              toast.success("Success", {
                description: projectOutput.message
              })
              deleteButtonState = false;
            } catch (/** @type {any} */ err) {
              ExceptionRef = err.message;
              toast.error("Error", {
                description: "Failed to initiate project deletion",
              })
            }
            deleteButtonState = false;
          }}>
            Delete Project
            {#if deleteButtonState}
              <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
            {/if}
          </Button>
        </Dialog.Description>
      </Dialog.Header>
    </Dialog.Content>
  </Dialog.Root>
</div>