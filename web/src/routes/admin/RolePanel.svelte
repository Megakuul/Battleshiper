<script>
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import * as Select from "$lib/components/ui/select";
  import { Button } from "$lib/components/ui/button";
  import { toast } from "svelte-sonner";
  import { Input } from "$lib/components/ui/input";
  import { UpdateRole } from "$lib/adapter/admin/updaterole";

  /** @type {ROLE[]} */
  export const ROLE_LIST = ["USER", "SUPPORT", "MAINTAINER", "SUBSCRIPTION_MANAGER", "ROLE_MANAGER"]

  /** @type {string} */
  export let ExceptionRef;

  /** @type {Object.<ROLE, null>}*/
  export let UserRoles;

  /** @type {import("$lib/adapter/admin/updaterole").updateRoleInput}*/
  let updateRoleInput = {
    user_id: "",
    rbac_roles: ""
  };

  /** @type {boolean}*/
  let updateButtonState;
</script>

<div class="flex flex-col gap-2 w-10/12 p-5 bg-slate-900/30 rounded-lg">
  <h1 class="text-2xl font-bold">Role</h1>
  {#if "ROLE_MANAGER" in UserRoles}
    <Input bind:value={updateRoleInput.user_id} type="text" placeholder="User ID" />
    <Select.Root multiple={true} onSelectedChange={(v) => {
      if (v) {
        updateRoleInput.rbac_roles = {};
        v.forEach((role) => {
          updateRoleInput.rbac_roles[role.value] = null;
        })
      } else {
        toast("Validation Exception", {
          description: "Failed to apply roles",
        })
      }
    }}>
      <Select.Trigger>
        <Select.Value placeholder="ROLES" />
      </Select.Trigger>
      <Select.Content>
        {#each ROLE_LIST as role}
          <Select.Item value="{role}">{role}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>

    <Button type="submit" on:click={async () => {
      try {
        updateButtonState = true;
        const userOutput = await UpdateRole(updateRoleInput);
        toast.success("Success", {
          description: userOutput.message
        })
        ExceptionRef = "";
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
        toast.error("Exception", {
          description: "User not found",
        })
      }
      updateButtonState = false;
    }}>
      Update Roles
      {#if updateButtonState}
        <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
      {/if}
    </Button>
  {:else}
    <Alert.Root variant="destructive" class="mt-4">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Forbidden</Alert.Title>
      <Alert.Description>You need the <b>ROLE_MANAGER</b> role to access this section.</Alert.Description>
    </Alert.Root>
  {/if}
</div>