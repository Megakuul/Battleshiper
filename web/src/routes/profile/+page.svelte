<script>
  import { FetchInfo } from "$lib/adapter/user/fetchinfo";
  import Skeleton from "$lib/components/ui/skeleton/skeleton.svelte";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button } from "$lib/components/ui/button";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import * as Avatar from "$lib/components/ui/avatar";
  import Icon from '@iconify/svelte';
  import { onMount } from "svelte";
  import { RegisterUser } from "$lib/adapter/user/registeruser";
  import { Authorize } from "$lib/adapter/auth/authorize";
    import { User } from "lucide-svelte";


  /** @type {import("$lib/adapter/user/fetchinfo").fetchInfoOutput}*/
  let UserInfo = {
    id: "12345",
    name: "Example User",
    roles: {
      ADMIN: null,
      USER: null
    },
    provider: "github",
    avatar_url: "https://example.com/avatar.jpg",
    subscription: {
      id: "sub_001",
      name: "Premium Plan",
      pipeline_specs: {
        daily_builds: 10,
        daily_deployments: 5
      },
      project_specs: {
        project_count: 3,
        alias_count: 2,
        prerender_routes: 100,
        server_storage: 5000,
        client_storage: 2000,
        prerender_storage: 3000
      },
      cdn_specs: {
        instance_count: 15
      }
    }
  };

  let Error = "";

  onMount(async () => {
    try {
      UserInfo = await FetchInfo();
    } catch (/** @type {any} */ err) {
      Error = err.message;
    }
  })
  
</script>

{#if UserInfo}
  <div class="flex flex-col gap-8 lg:flex-row my-20 min-h-[60vh] mx-12">
    <div class="p-12 rounded-lg w-full bg-slate-800 bg-opacity-55 flex flex-col justify-center items-center gap-4">
      <div class="flex flex-row justify-start items-start gap-4">
        <Avatar.Root class="h-20 w-20">
          <Avatar.Image src="{UserInfo.avatar_url}" alt="{UserInfo.name}" />
          <Avatar.Fallback>-</Avatar.Fallback>
        </Avatar.Root>
        <div class="flex flex-col">
          <h1 class="text-6xl opacity-80 font-bold">{UserInfo.name}</h1>
          <h2 class="text-3xl opacity-80 mb-4">#{UserInfo.id}</h2>
          <h2 class="text-3xl opacity-80"><span class="font-bold">Provider: </span>
            <span class="lowercase">{UserInfo.provider}</span>
          </h2>
          <h2 class="text-3xl opacity-80">
            <span class="font-bold">Roles: [</span>
              {#each Object.entries(UserInfo.roles) as role, i}
                <span class="lowercase">{i===0 ? "" : ","} {role[0]} </span>
              {/each}
            <span class="font-bold">]</span></h2>
        </div>
      </div>
    </div>
    <div class="p-12 rounded-lg w-full bg-slate-800 bg-opacity-55 flex flex-col justify-center items-center gap-4"> 
      {#if UserInfo.subscription}
        <h1 class="text-6xl opacity-80 font-bold">{UserInfo.subscription.name}</h1>
        <h2 class="text-3xl opacity-80 mb-4">#{UserInfo.subscription.id}</h2>
        <div class="w-10/12 bg-slate-700">
          <h3 class="text-2xl opacity-80">Project Specs</h3>
          <h2 class="text-xl opacity-80"><span class="font-bold">Projects: </span>
            <span>{UserInfo.subscription.project_specs.project_count}</span>
          </h2>
          <h2 class="text-xl opacity-80 w-full"><span class="font-bold">Allowed Aliases: </span>
            <span>{UserInfo.subscription.project_specs.alias_count}</span>
          </h2>
          <h2 class="text-xl opacity-80"><span class="font-bold">Prerender Routes: </span>
            <span>{UserInfo.subscription.project_specs.prerender_routes}</span>
          </h2>
          <h2 class="text-xl opacity-80"><span class="font-bold">Prerender Storage: </span>
            <span>{UserInfo.subscription.project_specs.prerender_storage}</span>
          </h2>
          <h2 class="text-xl opacity-80"><span class="font-bold">Client Storage: </span>
            <span>{UserInfo.subscription.project_specs.client_storage}</span>
          </h2>
          <h2 class="text-xl opacity-80"><span class="font-bold">Server Storage: </span>
            <span>{UserInfo.subscription.project_specs.server_storage}</span>
          </h2>
        </div>
      {:else}
        <h1 class="text-4xl opacity-80 font-bold">Subscription</h1>
        <Alert.Root class="mt-auto">
          <CircleAlert class="h-4 w-4" />
          <Alert.Title>Warning</Alert.Title>
          <Alert.Description>No subscription associated</Alert.Description>
        </Alert.Root>
      {/if}
    </div>
  </div>
{:else if Error}
  <div class="min-h-[60vh] flex justify-center items-center">
    <h1 class="text-6xl opacity-80">Oops... please log in to continue!</h1>
  </div>
  <div class="m-6 flex flex-col space-y-4">
    <Tooltip.Root>
      <Tooltip.Trigger asChild let:builder>
        <Button builders={[builder]} variant="outline" class="text-2xl h-20" on:click={Authorize}>
          LOG IN <Icon icon="line-md:github-loop" class="ml-2" />
        </Button>
      </Tooltip.Trigger>
      <Tooltip.Content>
        <p>Log in with Github</p>
      </Tooltip.Content>
    </Tooltip.Root>

    <Tooltip.Root>
      <Tooltip.Trigger asChild let:builder>
        <Button builders={[builder]} variant="outline" class="text-2xl h-20" on:click={RegisterUser}>
          REGISTER <Icon icon="line-md:person-add-twotone" class="ml-2" />
        </Button>
      </Tooltip.Trigger>
      <Tooltip.Content>
        <p>Not registered yet? Link your Github account to get started.</p>
      </Tooltip.Content>
    </Tooltip.Root>

    <Alert.Root variant="destructive">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Error}</Alert.Description>
    </Alert.Root>
  </div>
{:else}
  <div class="flex items-center justify-center space-x-4">
    <Skeleton class="h-36 w-36 rounded-full bg-slate-800" />
    <div class="space-y-2">
      <Skeleton class="h-8 w-[250px] bg-slate-800" />
      <Skeleton class="h-8 w-[200px] bg-slate-800" />
      <Skeleton class="h-8 w-[200px] bg-slate-800" />
    </div>
  </div>
{/if}
