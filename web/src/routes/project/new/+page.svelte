<script>
  import { goto } from "$app/navigation";
  import * as Alert from "$lib/components/ui/alert";
  import * as Select from "$lib/components/ui/select";
  import * as Popover from "$lib/components/ui/popover";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import Icon from "@iconify/svelte";
  import { CircleAlert } from "lucide-svelte";
  import { fade } from "svelte/transition";
  import { RepositoryInfo } from "$lib/stores";
  import { onMount } from "svelte";
  import { ListRepository } from "$lib/adapter/resource/listrepository";
  import { toast } from "svelte-sonner";
  import { CreateProject } from "$lib/adapter/resource/createproject";

  import {
    PUBLIC_BATTLESHIPER_APP_URL,
    PUBLIC_SEO_DOMAIN
  } from "$env/static/public"

  /** @type {import("$lib/adapter/resource/createproject").createProjectInput} */
  let CurrentProjectInput = {
    project_name: "",
    build_command: "npm run build",
    build_image: "megakuul/battleshiper-build:latest",
    output_directory: "./build",
    repository: {
      id: 0,
      branch: "main",
      url: "",
    }
  };

  /** @type {string} */
  let Exception = "";

  onMount(async () => {
    try {
      if (!$RepositoryInfo) {
        $RepositoryInfo = await ListRepository();
      }
      $RepositoryInfo.repositories[0].full_name
    } catch (/** @type {any} */ err) {
      Exception = err.message;
      toast("Error", {
        description: "Failed to load available repositories",
      })
    }
  })

  // Generate a user friendly message if the exception is longer then 250 chars.
  // This is primarely for unexpected errors that cause the api to return an error page in html format.
  $: if (Exception && Exception.length > 250) Exception = "Unexpected error occured";

  /** @type {boolean} */
  let createButtonState;

  /** 
   * @param {string} full_name
   * @returns {string}
   */
  function getRepositoryUrl(full_name) {
    return `https://github.com/${full_name}`;
  }
</script>

<svelte:head>
	<title>New Project | Battleshiper</title>
  <meta name="robots" content="noindex, follow">
	<meta name="description" content="Launch applications today and fulfill your vision." />
	<meta property="og:description" content="Launch applications today and fulfill your vision." />
	<meta property="og:title" content="New Project - Battleshiper">
  <meta property="og:type" content="website">
	<meta property="og:image" content="https://{PUBLIC_SEO_DOMAIN}/favicon.png" />
	<link rel="canonical" href="https://{PUBLIC_SEO_DOMAIN}/project/new" />
</svelte:head>

<div transition:fade class="flex flex-col gap-8 justify-center mt-12 mb-16">
  <h1 class="text-6xl font-bold text-center text-slate-200/80">New Project</h1>
</div>

<div transition:fade class="m-6 min-h-[60vh] flex justify-center">
  <div class="flex flex-col gap-8 w-11/12 lg:w-8/12 mb-10 p-10 rounded-lg bg-slate-900/20">
    <div class="flex flex-col sm:flex-row gap-4">
      <Input bind:value={CurrentProjectInput.project_name} type="text" placeholder="Project Name" class="" />
      <Input bind:value={CurrentProjectInput.repository.branch} type="text" placeholder="Branch" class="w-full sm:w-[80px]" />
      <Select.Root onSelectedChange={(v) => {
        if (v && v.value && v.value.id && v.value.full_name) {
          CurrentProjectInput.repository.id = v.value.id;
          CurrentProjectInput.repository.url = getRepositoryUrl(v.value.full_name);
        } else {
          toast("Validation Exception", {
            description: "Failed to apply repository",
          })
        }
      }}>
        <Select.Trigger class="w-full sm:w-[500px]">
          <Select.Value  placeholder="Repository" />
        </Select.Trigger>
        <Select.Content>
          {#if $RepositoryInfo && $RepositoryInfo.repositories}
            {#each $RepositoryInfo.repositories as repo}
              <Select.Item class="text-xs" value="{repo}">{repo?.full_name}</Select.Item>
            {/each}
          {/if}
        </Select.Content>
      </Select.Root>
      <Popover.Root>
        <Popover.Trigger class="hidden sm:block"><Icon icon="octicon:info-16" /></Popover.Trigger>
        <Popover.Content>
          Install the <a class="underline font-semibold" href="{PUBLIC_BATTLESHIPER_APP_URL}">Battleshiper</a> App,
          <br> and grant read access to the desired repositories.
        </Popover.Content>
      </Popover.Root>
    </div>
    <Input bind:value={CurrentProjectInput.build_image} type="text" placeholder="Build Image" />
    <Input bind:value={CurrentProjectInput.build_command} type="text" placeholder="Build Command" />
    <Input bind:value={CurrentProjectInput.output_directory} type="text" placeholder="Build Output Directory" />
    <Button class="mt-auto" type="submit" on:click={async () => {
      try {
        createButtonState = true;
        const projectOutput = await CreateProject(CurrentProjectInput);
        toast.success("Success", {
          description: projectOutput.message
        })
        createButtonState = false;
        goto("/project");
      } catch (/** @type {any} */ err) {
        Exception = err.message;
        toast.error("Exception", {
          description: "Failed to create project",
        })
      }
      createButtonState = false;
    }}>
      Create Project
      {#if createButtonState}
        <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
      {/if}
    </Button>
  </div>
</div>

{#if Exception}
  <div transition:fade class="m-6 flex flex-col items-center space-y-4">
    <Alert.Root variant="destructive" class="w-11/12 lg:w-8/12">
      <CircleAlert class="h-4 w-4" />
      <Alert.Title>Error</Alert.Title>
      <Alert.Description>{Exception}</Alert.Description>
    </Alert.Root>
  </div>
{/if}