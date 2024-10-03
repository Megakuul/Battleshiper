<script>
  import { FetchInfo } from "$lib/adapter/user/fetchinfo";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button } from "$lib/components/ui/button";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import * as Avatar from "$lib/components/ui/avatar";
  import Icon from '@iconify/svelte';
  import { onMount } from "svelte";
  import { Authorize } from "$lib/adapter/auth/authorize";
  import SpecItem from "./SpecItem.svelte";
  import { fade } from "svelte/transition";
  import { UserInfo } from "$lib/stores";
  import { toast } from "svelte-sonner";
  import { Logout } from "$lib/adapter/auth/logout";
  import { PUBLIC_SEO_DOMAIN } from "$env/static/public";
  import { RegisterUser } from "$lib/adapter/user/registeruser";

  /** @type {boolean} */
  let Registered = true;

  /** @type {string} */
  let Exception = "";

  onMount(async () => {
    try {
      if (!$UserInfo) {
        // Github suffers a major issue with refresh token (https://github.com/orgs/community/discussions/24745)
        // Therefore the refresh process is currently not employed, as it would run into errors very oftens.
        // await Refresh(); 
        $UserInfo = await FetchInfo();
      }
    } catch (/** @type {any} */ err) {
      if (err.statusCode === 404) {
        Registered = false;
      } else {
        Exception = err.message;
      }
    }
  })

  // Generate a user friendly message if the exception is longer then 250 chars.
  // This is primarely for unexpected errors that cause the api to return an error page in html format.
  $: if (Exception && Exception.length > 250) Exception = "Unexpected error occured";
  
  /** 
   * @param {number} bytes
   * @returns {number}
  */
  function bytesToGigabytes(bytes) {
    return parseFloat((bytes / 1000000000).toFixed(2))
  }
</script>

<svelte:head>
	<title>Profile | Battleshiper</title>
	<meta name="description" content="Check out your user profile and view your subscription status." />
	<meta property="og:description" content="Check out your user profile and view your subscription status." />
	<meta property="og:title" content="Profile - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}/profile" />
</svelte:head>

{#if $UserInfo}
  <div transition:fade class="flex flex-col lg:flex-row gap-8 my-20 min-h-[80vh] mx-6 md:mx-12">
    <div class="w-full bg-slate-800 bg-opacity-55 p-6 md:p-12 rounded-lg overflow-hidden flex flex-col justify-center items-center gap-4">
      <div class="flex flex-row justify-start items-start gap-4">
        <Avatar.Root class="h-16 md:h-20 w-16 md:w-20">
          <Avatar.Image src="{$UserInfo.avatar_url}" alt="{$UserInfo.name}" />
          <Avatar.Fallback>-</Avatar.Fallback>
        </Avatar.Root>
        <div class="flex flex-col">
          <h1 class="text-3xl md:text-6xl opacity-80 font-bold">{$UserInfo.name}</h1>
          <h2 class="text-xl md:text-3xl opacity-80 mb-4">#{$UserInfo.id}</h2>
          <h2 class="text-xl md:text-3xl opacity-80"><span class="font-bold">Provider: </span>
            <span class="lowercase text-[rgba(132,62,35,1)]">"{$UserInfo.provider}"</span>
          </h2>
          <h2 class="text-xl md:text-3xl opacity-80">
            <span class="font-bold block">Roles: <span class="text-yellow-700">[</span></span>
              {#each Object.entries($UserInfo.roles) as role}
                <span class="lowercase ml-4 block text-[rgba(132,62,35,1)]">"{role[0]}",</span>
              {/each}
            <span class="font-bold block text-yellow-700">]</span>
          </h2>
        </div>
      </div>
      <Button variant="secondary" class="text-xl md:text-2xl" on:click={async () => {
        try {
          await Logout();
          toast.success("Success", {
            description: "Successfully logged out"
          })
          $UserInfo = undefined;
          Exception = "User not logged in";
        } catch (/** @type {any} */ err) {
          Exception = err.message;
          toast.error("Error", {
            description: "Failed to log out",
          })
        }
      }}>
        LOGOUT <Icon icon="line-md:logout" class="ml-2" />
      </Button>
    </div>

    <div class="w-full bg-slate-800 bg-opacity-55 rounded-lg p-12 max-h-[80vh] overflow-scroll-hidden flex flex-col items-center gap-4"> 
      {#if $UserInfo.subscription}
        <h1 class="text-3xl md:text-6xl opacity-80 font-bold">{$UserInfo.subscription.name}</h1>
        <h2 class="text-xl md:text-3xl opacity-80 mb-4">#{$UserInfo.subscription.id}</h2>
        {#if $UserInfo.subscription.project_specs}
        <div class="w-full rounded-xl bg-slate-800 bg-opacity-40 p-8">
          <h3 class="text-2xl font-bold opacity-80 text-center mb-10">Project Specs</h3>
          <div class="flex flex-wrap justify-around items-center gap-6">
            <SpecItem 
              title="Projects" 
              value="{$UserInfo.subscription.project_specs.project_count}x" 
              description="{$UserInfo.subscription.project_specs.project_count} projects">
            </SpecItem>
            <SpecItem 
              title="Aliases" 
              value="{$UserInfo.subscription.project_specs.alias_count}x" 
              description="{$UserInfo.subscription.project_specs.alias_count} aliases per project">
            </SpecItem>
            <SpecItem 
              title="Prerender Routes" 
              value="{$UserInfo.subscription.project_specs.prerender_routes}x" 
              description="{$UserInfo.subscription.project_specs.prerender_routes} prerender routes per project">
            </SpecItem>
            <SpecItem 
              title="Prerender Storage" 
              value="{bytesToGigabytes($UserInfo.subscription.project_specs.prerender_storage)}GB" 
              description="{bytesToGigabytes($UserInfo.subscription.project_specs.prerender_storage)} GB per project">
            </SpecItem>
            <SpecItem 
              title="Client Storage" 
              value="{bytesToGigabytes($UserInfo.subscription.project_specs.client_storage)}GB" 
              description="{bytesToGigabytes($UserInfo.subscription.project_specs.client_storage)} GB per project">
            </SpecItem>
            <SpecItem 
              title="Server Storage" 
              value="{bytesToGigabytes($UserInfo.subscription.project_specs.server_storage)}GB" 
              description="{bytesToGigabytes($UserInfo.subscription.project_specs.server_storage)} GB per project">
            </SpecItem>
          </div>
        </div>
        {/if}
        {#if $UserInfo.subscription.pipeline_specs}
        <div class="w-full rounded-xl bg-slate-800 bg-opacity-40 p-8">
          <h3 class="text-2xl font-bold opacity-80 text-center mb-10">Pipeline Specs</h3>
          <div class="flex flex-wrap justify-around items-center gap-6">
            <SpecItem 
              title="Daily Builds" 
              value="{$UserInfo.subscription.pipeline_specs.daily_builds}x" 
              description="{$UserInfo.subscription.pipeline_specs.daily_builds} builds per day">
            </SpecItem>
            <SpecItem 
              title="Daily Deployments" 
              value="{$UserInfo.subscription.pipeline_specs.daily_deployments}x" 
              description="{$UserInfo.subscription.pipeline_specs.daily_deployments} deployments per day">
            </SpecItem>
          </div>
        </div>
        {/if}
        {#if $UserInfo.subscription.cdn_specs}
        <div class="w-full rounded-xl bg-slate-800 bg-opacity-40 p-8">
          <h3 class="text-2xl font-bold opacity-80 text-center mb-10">CDN Specs</h3>
          <div class="flex flex-wrap justify-around items-center gap-6">
            <SpecItem 
              title="CDN Instances" 
              value="{$UserInfo.subscription.cdn_specs.instance_count}x" 
              description="{$UserInfo.subscription.cdn_specs.instance_count} instances">
            </SpecItem>
          </div>
        </div>
        {/if}
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
{:else if !Registered}
  <div transition:fade class="min-h-[60vh] flex justify-center items-center">
    <h1 class="text-3xl md:text-6xl text-center opacity-80">Not so fast! Registration is required!</h1>
  </div>
  <div transition:fade class="m-6 flex flex-col space-y-4">
    <Tooltip.Root>
      <Tooltip.Trigger asChild let:builder>
        <Button builders={[builder]} variant="outline" class="text-2xl h-20" on:click={async () => {
          try {
            await RegisterUser()
            toast.success("Success", {
              description: "User registered"
            })
            Registered = true;
            $UserInfo = await FetchInfo();
            Exception = "";
          } catch (/** @type {any} */ err) {
            Exception = err.message;
            toast.error("Error", {
              description: "Failed register user",
            })
          }
        }}>
          REGISTER <Icon icon="line-md:person-add-twotone" class="ml-2" />
        </Button>
      </Tooltip.Trigger>
      <Tooltip.Content>
        <p>Not registered yet? Link your Github account to get started.</p>
      </Tooltip.Content>
    </Tooltip.Root>
  </div>
{:else if Exception}
  <div transition:fade class="min-h-[60vh] flex justify-center items-center">
    <h1 class="text-3xl md:text-6xl text-center opacity-80">Oops... please log in to continue!</h1>
  </div>
  <div transition:fade class="m-6 flex flex-col space-y-4">
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

    <Alert.Root variant="destructive">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Exception}</Alert.Description>
    </Alert.Root>
  </div>
{:else}
  <div transition:fade class="min-h-[80vh] flex justify-center items-center">
    <LoaderCircle class="mr-2 h-16 w-16 opacity-80 animate-spin" />
  </div>
{/if}