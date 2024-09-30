<script>
  import { FetchInfo } from "$lib/adapter/user/fetchinfo";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import CircleAlert from "lucide-svelte/icons/circle-alert";
  import * as Alert from "$lib/components/ui/alert/index.js";
  import { Button } from "$lib/components/ui/button";
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import Icon from '@iconify/svelte';
  import { onMount } from "svelte";
  import { Authorize } from "$lib/adapter/auth/authorize";
  import { fade } from "svelte/transition";
  import { UserInfo } from "$lib/stores";
  import { 
    PUBLIC_SEO_DOMAIN 
  } from "$env/static/public";
  import UserPanel from "./UserPanel.svelte";
  import ProjectPanel from "./ProjectPanel.svelte";
  import SubscriptionPanel from "./SubscriptionPanel.svelte";
  import RolePanel from "./RolePanel.svelte";

  /** @type {string} */
  let Exception = "";
  
  onMount(async () => {
    try {
      if (!$UserInfo) {
        $UserInfo = await FetchInfo();
      }
    } catch (/** @type {any} */ err) {
      Exception = err.message;
    }
  })

  // Generate a user friendly message if the exception is longer then 250 chars.
  // This is primarely for unexpected errors that cause the api to return an error page in html format.
  $: if (Exception && Exception.length > 250) Exception = "Unexpected error occured";
</script>

<svelte:head>
	<title>Admin | Battleshiper</title>
  <meta name="description" content="Manage Battleshiper users, projects and permissions in one place." />
	<meta property="og:description" content="Manage Battleshiper users, projects and permissions in one place." />
	<meta property="og:title" content="Admin - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}/admin" />
</svelte:head>

{#if $UserInfo}
  <div class="flex flex-col gap-8 items-center mt-12 mb-16">
    <h1 class="text-6xl font-bold text-center text-slate-200/80 ">Admin Center</h1>

    <UserPanel bind:ExceptionRef={Exception} UserRoles={$UserInfo.roles} />

    <ProjectPanel bind:ExceptionRef={Exception} UserRoles={$UserInfo.roles} />

    <SubscriptionPanel bind:ExceptionRef={Exception} UserRoles={$UserInfo.roles} />

    <RolePanel bind:ExceptionRef={Exception} UserRoles={$UserInfo.roles} />
  </div>
{:else if Exception}
  <div transition:fade class="min-h-[60vh] flex justify-center items-center">
    <h1 class="text-3xl md:text-6xl text-center opacity-80">Yikes! Admin Center ran into a hiccup.</h1>
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