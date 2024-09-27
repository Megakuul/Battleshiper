<script>
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import * as Dialog from "$lib/components/ui/dialog";
  import Icon from '@iconify/svelte';
  import { fade } from "svelte/transition";
  import { Input } from "$lib/components/ui/input";
  import { toast } from "svelte-sonner";
  import { FindUser } from "$lib/adapter/admin/finduser";
  import { cn } from "$lib/utils";
  import { DeleteUser } from "$lib/adapter/admin/deleteuser";
  import { UpdateUser } from "$lib/adapter/admin/updateuser";


  /** @type {string} */
  export let ExceptionRef;

  /** @type {Object.<ROLE, null>}*/
  export let UserRoles;

  /** @type {import("$lib/adapter/admin/finduser").findUserInput}*/
  let findUserInput = {
    user_id: "",
  }

  /** @type {import("$lib/adapter/admin/finduser").findUserOutput}*/
  let findUserOutput;

  /** @type {import("$lib/adapter/admin/updateuser").updateUserInput}*/
  let updateUserInput = {
    user_id: "",
    update: {
      subscription_id: "",
    }
  };

  /** @type {boolean}*/
  let findButtonState;

  /** @type {boolean}*/
  let updateButtonState;

  /** @type {boolean}*/
  let deleteButtonState;
</script>

<div class="flex flex-col gap-2 w-10/12 p-5 bg-slate-900/30 rounded-lg">
  <h1 class="text-2xl font-bold">Users</h1>
  {#if "SUPPORT" in UserRoles || "MAINTAINER" in UserRoles}
    <Input bind:value={findUserInput.user_id} type="text" placeholder="User ID" />
    <Button type="submit" on:click={async () => {
      try {
        findButtonState = true;
        findUserOutput = await FindUser(findUserInput);
        toast.success("Success", {
          description: findUserOutput.message
        })
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
        toast.error("Exception", {
          description: "User not found",
        })
      }
      findButtonState = false;
    }}>
      Find User
      {#if findButtonState}
        <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
      {/if}
    </Button>
    {#if findUserOutput && findUserOutput.user}
    <div transition:fade class="flex flex-col gap-2 max-h-[40yyvh] mt-4 p-3 bg-slate-600/20 rounded-md overflow-scroll-hidden">
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-16 sm:w-32 text-xs sm:text-base font-bold">User ID</p>
        <p class="w-24 sm:w-32 text-xs sm:text-base font-bold mr-auto">Subscription ID</p>
      </div>
      <hr>
      {#each [findUserOutput.user] as user}
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-16 sm:w-32 text-xs sm:text-base overflow-scroll-hidden">{user.id}</p>
        <p class="w-24 sm:w-32 text-xs sm:text-base overflow-scroll-hidden mr-auto">{user.subscription_id}</p>
        <Dialog.Root>
          <Dialog.Trigger >
            <Button variant="ghost" on:click={() => updateUserInput.user_id = user.id}>
              <Icon icon="line-md:edit-full-twotone" class="h-4 sm:h-6 w-4 sm:w-6" />
            </Button>
          </Dialog.Trigger>
          <Dialog.Content>
            <Dialog.Header>
              <Dialog.Title class="text-center mb-6">Update User</Dialog.Title>
              <Dialog.Description class="flex flex-col gap-2">
                <Input bind:value={updateUserInput.user_id} type="text" placeholder="User ID" />
                <Input bind:value={updateUserInput.update.subscription_id} type="text" placeholder="Subscription ID" />
                
                <Button class="w-full mt-6" type="submit" on:click={async () => {
                  try {
                    updateButtonState = true;
                    const userOutput = await UpdateUser(updateUserInput)
                    toast.success("Success", {
                      description: userOutput.message
                    })
                    updateButtonState = false;
                  } catch (/** @type {any} */ err) {
                    ExceptionRef = err.message;
                    toast.error("Error", {
                      description: "Failed to update user",
                    })
                  }
                  updateButtonState = false;
                }}>
                  Update User
                  {#if updateButtonState}
                    <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
                  {/if}
                </Button>
              </Dialog.Description>
            </Dialog.Header>
          </Dialog.Content>
        </Dialog.Root>
        {#if user.privileged}
          <Tooltip.Root>
            <Tooltip.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="ic:twotone-lock" class="h-4 sm:h-6 w-4 sm:w-6" /></Tooltip.Trigger>
            <Tooltip.Content>
              <p>User is privileged</p>
            </Tooltip.Content>
          </Tooltip.Root>
        {:else}
          <Dialog.Root>
            <Dialog.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="line-md:person-remove-twotone" class="h-4 sm:h-6 w-4 sm:w-6" /></Dialog.Trigger>
            <Dialog.Content>
              <Dialog.Header>
                <Dialog.Title class="text-center">Delete user '{user.id}'?</Dialog.Title>
                <Dialog.Description>
                  <Button variant="destructive" class="w-full mt-6" type="submit" on:click={async () => {
                    try {
                      deleteButtonState = true;
                      const userOutput = await DeleteUser({
                        user_id: user.id,
                      })
                      toast.success("Success", {
                        description: userOutput.message
                      })
                      deleteButtonState = false;
                    } catch (/** @type {any} */ err) {
                      ExceptionRef = err.message;
                      toast.error("Error", {
                        description: "Failed to delete user",
                      })
                    }
                    deleteButtonState = false;
                  }}>
                    Delete User
                    {#if deleteButtonState}
                      <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
                    {/if}
                  </Button>
                </Dialog.Description>
              </Dialog.Header>
            </Dialog.Content>
          </Dialog.Root>
        {/if}
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