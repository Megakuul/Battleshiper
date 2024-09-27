<script>
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";
  import * as Popover from "$lib/components/ui/popover";
  import Icon from '@iconify/svelte';
  import { fade } from "svelte/transition";
  import { Input } from "$lib/components/ui/input";
  import { toast } from "svelte-sonner";
  import { cn } from "$lib/utils";
  import { FindProject } from "$lib/adapter/admin/findproject";
  import { DeleteProject } from "$lib/adapter/admin/deleteproject";


  /** @type {string} */
  export let ExceptionRef;

  /** @type {Object.<ROLE, null>}*/
  export let UserRoles;

  /** @type {import("$lib/adapter/admin/findproject").findProjectInput}*/
  let findProjectInput = {
    project_name: "",
    owner_id: ""
  }

  /** @type {import("$lib/adapter/admin/findproject").findProjectOutput}*/
  let findProjectOutput;

  /** @type {boolean}*/
  let findButtonState;

  /** @type {boolean}*/
  let deleteButtonState;
</script>

<div class="flex flex-col gap-2 w-10/12 p-5 bg-slate-900/30 rounded-lg">
  <h1 class="text-2xl font-bold">Projects</h1>
  {#if "SUPPORT" in UserRoles || "MAINTAINER" in UserRoles}
    <div class="flex flex-col sm:flex-row gap-2">
      <Input bind:value={findProjectInput.project_name} type="text" placeholder="Project Name" />
      <Input bind:value={findProjectInput.owner_id} type="text" placeholder="Owner ID" />
    </div>
    <Button type="submit" on:click={async () => {
      try {
        findButtonState = true;
        findProjectOutput = await FindProject(findProjectInput);
        toast.success("Success", {
          description: findProjectOutput.message
        })
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
        toast.error("Exception", {
          description: "Project not found",
        })
      }
      findButtonState = false;
    }}>
      Find Projects
      {#if findButtonState}
        <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
      {/if}
    </Button>
    {#if findProjectOutput && findProjectOutput.projects}
    <div transition:fade class="flex flex-col gap-2 max-h-[40vh] mt-4 p-3 bg-slate-600/20 rounded-md overflow-scroll-hidden">
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-16 sm:w-52 text-xs sm:text-base font-bold">Name</p>
        <p class="w-24 sm:w-32 text-xs sm:text-base font-bold mr-auto">Owner ID</p>
      </div>
      <hr>
      {#each findProjectOutput.projects as project}
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-16 sm:w-52 text-xs sm:text-base overflow-scroll-hidden">{project.name}</p>
        <p class="w-24 sm:w-32 text-xs sm:text-base overflow-scroll-hidden mr-auto">{project.owner_id}</p>
        <Popover.Root>
          <Popover.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="line-md:chat-twotone" class="h-4 sm:h-6 w-4 sm:w-6" /></Popover.Trigger>
          <Popover.Content class="relative flex flex-col gap-2 p-4 items-center w-max">
            {#if project.deleted}
            <div class="absolute top-3 right-3 w-4 h-4 rounded-full bg-red-900"></div>
            {:else if !project.initialized}
            <div class="absolute top-3 right-3 w-4 h-4 rounded-full bg-blue-600"></div>
            {:else}
            <div class="absolute top-3 right-3 w-4 h-4 rounded-full bg-green-800"></div>
            {/if}
            <div class="flex flex-col gap-2 w-64 sm:w-96">
              <h1 class="text-sm sm:text-lg font-bold">
                Repository: 
              </h1>
              <p class="text-xs sm:text-lg p-2 rounded-lg break-all bg-slate-600/20 overflow-hidden">
                {project.repository.url}@{project.repository.branch}
              </p>
              <h1 class="text-sm sm:text-lg font-bold">
                Aliases: 
              </h1>
              <p class="text-xs sm:text-lg p-2 rounded-lg break-all bg-slate-600/20 overflow-hidden">
                <span class="font-bold">[</span>
                  {#each Object.entries(project.aliases) as alias}
                    <span>{alias[0]}, </span>
                  {/each}
                <span class="font-bold">]</span>
              </p>
              <h1 class="text-sm sm:text-lg font-bold">
                Status: 
                {#if !project.status}
                  <span class="text-green-700 uppercase block md:inline">System operational</span>
                {:else}
                  <span class="text-red-700 uppercase block md:inline">{project.status.slice(0, project.status.indexOf(":"))}</span>
                {/if}
              </h1>
              <p class="text-xs sm:text-lg text-red-700 p-2 rounded-lg break-all bg-slate-600/20 max-h-[10vh] overflow-scroll-hidden">
                {project.status.slice(project.status.indexOf(":") + 1)}
              </p>
            </div>
          </Popover.Content>
        </Popover.Root>
        <Dialog.Root>
          <Dialog.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="line-md:minus-circle" class="h-4 sm:h-6 w-4 sm:w-6" /></Dialog.Trigger>
          <Dialog.Content>
            <Dialog.Header>
              <Dialog.Title class="text-center">Delete project '{project.name}'?</Dialog.Title>
              <Dialog.Description>
                <Button variant="destructive" class="w-full mt-6" type="submit" on:click={async () => {
                  try {
                    deleteButtonState = true;
                    const userOutput = await DeleteProject({
                      project_name: project.name,
                    })
                    toast.success("Success", {
                      description: userOutput.message
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
      {/each}
    </div>
    {/if}
  {:else}
    <Alert.Root variant="destructive" class="mt-4">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Forbidden</Alert.Title>
      <Alert.Description>You need the <b>SUPPORT</b> or <b>MAINTAINER</b> role to access this section.</Alert.Description>
    </Alert.Root>
  {/if}
</div>