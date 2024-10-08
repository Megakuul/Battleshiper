<script>
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import * as Dialog from "$lib/components/ui/dialog";
  import * as Popover from "$lib/components/ui/popover";
  import Icon from '@iconify/svelte';
  import { fade } from "svelte/transition";
  import { toast } from "svelte-sonner";
  import { cn } from "$lib/utils";
  import { ListSubscription } from "$lib/adapter/admin/listsubscription";
  import { Input } from "$lib/components/ui/input";
  import { UpsertSubscription } from "$lib/adapter/admin/upsertsubscription";


  /** @type {string} */
  export let ExceptionRef;

  /** @type {Object.<ROLE, null>}*/
  export let UserRoles;

  /** @type {import("$lib/adapter/admin/listsubscription").listSubscriptionOutput}*/
  let listSubscriptionOutput;

  /** @type {import("$lib/adapter/admin/upsertsubscription").upsertSubscriptionInput}*/
  let upsertSubscriptionInput = {
    id: "",
    name: "",
    project_specs: {
      project_count: NaN,
      alias_count: NaN,
      prerender_routes: NaN,
      prerender_storage: NaN,
      client_storage: NaN,
      server_storage: NaN
    },
    pipeline_specs: {
      daily_builds: NaN,
      daily_deployments: NaN,
    },
    cdn_specs: {
      instance_count: NaN,
    }
  };

  /** @type {boolean}*/
  let listButtonState;

  /** @type {boolean}*/
  let upsertButtonState;

  /** 
   * @param {number} bytes
   * @returns {number}
  */
  function bytesToGigabytes(bytes) {
    return parseFloat((bytes / 1000000000).toFixed(2))
  }

  /**
   * @param {Event} e 
   * @returns {number}
  */
  function parseInputNumber(e) {
    if (e.target instanceof HTMLInputElement) {
      return parseInt(e.target.value);
    }
    return NaN;
  }
</script>

<div class="flex flex-col gap-2 w-10/12 p-5 bg-slate-900/30 rounded-lg">
  <h1 class="text-2xl font-bold">Subscriptions</h1>
  {#if "SUBSCRIPTION_MANAGER" in UserRoles}
    <Dialog.Root>
      <Dialog.Trigger class="{cn(buttonVariants({variant: "secondary"}))}">Upsert Subscription</Dialog.Trigger>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title class="text-center mb-6">Upsert Subscription</Dialog.Title>
          <Dialog.Description class="flex flex-col gap-2">
            <Input bind:value={upsertSubscriptionInput.id} type="text" placeholder="Subscription ID" />
            <Input bind:value={upsertSubscriptionInput.name} type="text" placeholder="Subscription Name" />
            <h1 class="text-sm sm:text-lg font-bold">
              Project Specs: 
            </h1>
            <Input type="number" placeholder="Projects"
              value={upsertSubscriptionInput.project_specs.project_count} 
              on:input={(e) => upsertSubscriptionInput.project_specs.project_count = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Aliases"
              value={upsertSubscriptionInput.project_specs.alias_count} 
              on:input={(e) => upsertSubscriptionInput.project_specs.alias_count = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Prerender Routes"
              value={upsertSubscriptionInput.project_specs.prerender_routes} 
              on:input={(e) => upsertSubscriptionInput.project_specs.prerender_routes = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Prerender Storage (bytes)"
              value={upsertSubscriptionInput.project_specs.prerender_storage} 
              on:input={(e) => upsertSubscriptionInput.project_specs.prerender_storage = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Client Storage (bytes)"
              value={upsertSubscriptionInput.project_specs.client_storage} 
              on:input={(e) => upsertSubscriptionInput.project_specs.client_storage = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Server Storage (bytes)"
              value={upsertSubscriptionInput.project_specs.server_storage} 
              on:input={(e) => upsertSubscriptionInput.project_specs.server_storage = parseInputNumber(e)}>
            </Input>
            <h1 class="text-sm sm:text-lg font-bold">
              Pipeline Specs:
            </h1>
            <Input type="number" placeholder="Daily Builds"
              value={upsertSubscriptionInput.pipeline_specs.daily_builds} 
              on:input={(e) => upsertSubscriptionInput.pipeline_specs.daily_builds = parseInputNumber(e)}>
            </Input>
            <Input type="number" placeholder="Daily Deployments"
              value={upsertSubscriptionInput.pipeline_specs.daily_deployments} 
              on:input={(e) => upsertSubscriptionInput.pipeline_specs.daily_deployments = parseInputNumber(e)}>
            </Input>
            <h1 class="text-sm sm:text-lg font-bold">
              CDN Specs:
            </h1>
            <Input disabled type="number" placeholder="CDN Instances (not implemented)"
              value={upsertSubscriptionInput.cdn_specs.instance_count} 
              on:input={(e) => upsertSubscriptionInput.cdn_specs.instance_count = parseInputNumber(e)}>
            </Input>
            
            <Button class="w-full mt-6" type="submit" on:click={async () => {
              try {
                upsertButtonState = true;
                const subscriptionOutput = await UpsertSubscription(upsertSubscriptionInput)
                toast.success("Success", {
                  description: subscriptionOutput.message
                })
                upsertButtonState = false;
                ExceptionRef = "";
              } catch (/** @type {any} */ err) {
                ExceptionRef = err.message;
                toast.error("Error", {
                  description: "Failed to upsert subscription",
                })
              }
              upsertButtonState = false;
            }}>
              Upsert Subscription
              {#if upsertButtonState}
                <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
              {/if}
            </Button>
          </Dialog.Description>
        </Dialog.Header>
      </Dialog.Content>
    </Dialog.Root>
    <Button type="submit" class="mt-2" on:click={async () => {
      try {
        listButtonState = true;
        listSubscriptionOutput = await ListSubscription();
        toast.success("Success", {
          description: listSubscriptionOutput.message
        })
      } catch (/** @type {any} */ err) {
        ExceptionRef = err.message;
        toast.error("Exception", {
          description: "Failed to list subscriptions",
        })
      }
      listButtonState = false;
    }}>
      List Subscriptions
      {#if listButtonState}
        <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
      {/if}
    </Button>
    {#if listSubscriptionOutput && listSubscriptionOutput.subscriptions}
    <div transition:fade class="flex flex-col gap-2 max-h-[40yyvh] mt-4 p-3 bg-slate-600/20 rounded-md overflow-scroll-hidden">
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-24 sm:w-52 text-xs sm:text-base font-bold">Subscription ID</p>
        <p class="w-24 sm:w-52 text-xs sm:text-base font-bold mr-auto">Subscription Name</p>
      </div>
      <hr>
      {#each listSubscriptionOutput.subscriptions as subscription}
      <div class="flex flex-row gap-2 justify-between items-center text-nowrap">
        <p class="w-24 sm:w-52 text-xs sm:text-base overflow-scroll-hidden">{subscription.id}</p>
        <p class="w-24 sm:w-52 text-xs sm:text-base overflow-scroll-hidden mr-auto">{subscription.name}</p>
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Button variant="ghost" on:click={() => upsertSubscriptionInput = subscription}>
              <Icon icon="line-md:upload-twotone-loop" class="h-4 sm:h-6 w-4 sm:w-6" />
            </Button>
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Add to upsert dialog</p>
          </Tooltip.Content>
        </Tooltip.Root>
        <Popover.Root>
          <Popover.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="line-md:chat-twotone" class="h-4 sm:h-6 w-4 sm:w-6" /></Popover.Trigger>
          <Popover.Content class="relative flex flex-col gap-2 p-4 items-center w-max">
            <div class="flex flex-col gap-2 w-64 sm:w-96">
              <h1 class="text-sm sm:text-lg font-bold">
                Project Specs: 
              </h1>
              <section class="text-xs sm:text-lg p-2 rounded-lg break-all bg-slate-600/20 overflow-hidden">
                <p><b>Projects: </b>{subscription.project_specs.project_count}x</p>
                <p><b>Aliases: </b>{subscription.project_specs.alias_count}x</p>
                <p><b>Prerender Routes: </b>{subscription.project_specs.prerender_routes}x</p>
                <p><b>Prerender Storage: </b>{bytesToGigabytes(subscription.project_specs.prerender_storage)}GB</p>
                <p><b>Client Storage: </b>{bytesToGigabytes(subscription.project_specs.client_storage)}GB</p>
                <p><b>Server Storage: </b>{bytesToGigabytes(subscription.project_specs.server_storage)}GB</p>
              </section>
              <h1 class="text-sm sm:text-lg font-bold">
                Pipeline Specs:
              </h1>
              <section class="text-xs sm:text-lg p-2 rounded-lg break-all bg-slate-600/20 overflow-hidden">
                <p><b>Builds: </b>{subscription.pipeline_specs.daily_builds}x</p>
                <p><b>Deployments: </b>{subscription.pipeline_specs.daily_deployments}x</p>
              </section>
              <h1 class="text-sm sm:text-lg font-bold">
                CDN Specs:
              </h1>
              <section class="text-xs sm:text-lg p-2 rounded-lg break-all bg-slate-600/20 overflow-hidden">
                <p><b>Instances: </b>{subscription.cdn_specs.instance_count}x</p>
              </section>
            </div>
          </Popover.Content>
        </Popover.Root>
      </div>
      {/each}
    </div>
    {/if}
  {:else}
    <Alert.Root variant="destructive" class="mt-4">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Forbidden</Alert.Title>
      <Alert.Description>You need the <b>SUBSCRIPTION_MANAGER</b> role to access this section.</Alert.Description>
    </Alert.Root>
  {/if}
</div>