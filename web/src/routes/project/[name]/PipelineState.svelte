<script>
  import ConnectionBeam from "$lib/components/ConnectionBeam.svelte";

  /** @type {import("$lib/adapter/resource/listproject").projectOutput}*/
  export let CurrentProjectRef;

  let baseContainer;

  let eventContainer;

  let buildContainer;

  let deployContainer;


  /**
   * Determines if the build pipeline step is in "processing" state
   * based on the execution identifier and the state of the other pipeline steps.
   * @param {import("$lib/adapter/resource/listproject").projectOutput} project
   * @returns {boolean}
   */
  function isBuildProcessing(project) {
    const eventExecIdentifier = project.last_event_result.execution_identifier;
    const buildExecIdentifier = project.last_build_result.execution_identifier;
    if (eventExecIdentifier != buildExecIdentifier && project.last_event_result.successful) {
      return true
    }
    return false
  }

  /**
   * Determines if the deploy pipeline step is in "processing" state
   * based on the execution identifier and the state of the other pipeline steps.
   * @param {import("$lib/adapter/resource/listproject").projectOutput} project
   * @returns {boolean}
   */
  function isDeployProcessing(project) {
    const eventExecIdentifier = project.last_event_result.execution_identifier;
    const buildExecIdentifier = project.last_build_result.execution_identifier;
    const deployExecIdentifier = project.last_build_result.execution_identifier;
    if (eventExecIdentifier != deployExecIdentifier && project.last_event_result.successful) {
      return true
    } else if (buildExecIdentifier != deployExecIdentifier && project.last_build_result.successful) {
      return true
    }
    return false
  }
</script>

<div class="flex flex-col gap-8 w-10/12 my-8">
  <div class="relative flex flex-col items-start gap-4 w-full p-6 rounded-lg overflow-hidden bg-slate-700/20">
    <h1 class="text-xl md:text-2xl font-bold">Pipeline</h1>

    <div bind:this={baseContainer} class="w-full flex flex-col gap-4 lg:flex-row justify-between items-center p-6 z-10">
      <div bind:this={eventContainer} class="w-full lg:w-1/4 h-60 flex flex-col gap-2 items-center bg-slate-950 p-4 rounded-lg border-[1px] border-slate-200/15">
        <h1 class="text-xl font-bold">Event</h1>
        <h2 class="text-xs text-center">{CurrentProjectRef.last_event_result.execution_identifier}</h2>

        <div class="w-full h-full flex flex-col justify-center items-center m-2 border-[1px] rounded-lg">
          {#if CurrentProjectRef.last_event_result.successful}
            <p class="text-xl font-bold text-green-800">SUCCESS</p>
          {:else}
            <p class="text-xl font-bold text-red-800">FAILURE</p>
          {/if} 
          <p class="text-sm text-opacity-60">{new Date(CurrentProjectRef.last_event_result.timestamp).toLocaleString("en-US", {
            minute: "2-digit",
            hour: "2-digit",
            hour12: false,
            day: "2-digit",
            month: "2-digit",
            year: "numeric"
          })}</p>
        </div>
      </div>

      <div bind:this={buildContainer} class="w-full lg:w-1/4 h-60 flex flex-col gap-2 items-center bg-slate-950 p-4 rounded-lg border-[1px] border-slate-200/15">
        <h1 class="text-xl font-bold text-center">Build</h1>
        <h2 class="text-xs text-center">{CurrentProjectRef.last_build_result.execution_identifier}</h2>

        <div class="w-full h-full flex flex-col justify-center items-center m-2 border-[1px] rounded-lg">
          {#if isBuildProcessing(CurrentProjectRef)}
            <p class="text-xl font-bold text-orange-600">PROCESSING</p>
          {:else}
            {#if CurrentProjectRef.last_build_result.successful}
              <p class="text-xl font-bold text-green-800">SUCCESS</p>
            {:else}
              <p class="text-xl font-bold text-red-800">FAILURE</p>
            {/if}
            <p class="text-sm text-opacity-60">{new Date(CurrentProjectRef.last_build_result.timestamp).toLocaleString("en-US", {
              minute: "2-digit",
              hour: "2-digit",
              hour12: false,
              day: "2-digit",
              month: "2-digit",
              year: "numeric"
            })}</p>
          {/if}
        </div>
      </div>

      <div bind:this={deployContainer} class="w-full lg:w-1/4 h-60 flex flex-col gap-2 items-center bg-slate-950 p-4 rounded-lg border-[1px] border-slate-200/15">
        <h1 class="text-xl font-bold text-center">Deploy</h1>
        <h2 class="text-xs text-center">{CurrentProjectRef.last_deployment_result.execution_identifier}</h2>

        <div class="w-full h-full flex flex-col justify-center items-center m-2 border-[1px] rounded-lg">
          {#if isDeployProcessing(CurrentProjectRef)}
            <p class="text-xl font-bold text-orange-600">PROCESSING</p>
          {:else}
            {#if CurrentProjectRef.last_deployment_result.successful}
              <p class="text-xl font-bold text-green-800">SUCCESS</p>
            {:else}
              <p class="text-xl font-bold text-red-800">FAILURE</p>
            {/if}
            <p class="text-sm text-opacity-60">{new Date(CurrentProjectRef.last_deployment_result.timestamp).toLocaleString("en-US", {
              minute: "2-digit",
              hour: "2-digit",
              hour12: false,
              day: "2-digit",
              month: "2-digit",
              year: "numeric"
            })}</p>
          {/if}
        </div>
      </div>
    </div>
    
    <ConnectionBeam duration={3} 
      bind:containerRef={baseContainer}
      bind:fromRef={eventContainer}
      bind:toRef={buildContainer}
    />

    <ConnectionBeam duration={3} 
      bind:containerRef={baseContainer}
      bind:fromRef={buildContainer}
      bind:toRef={deployContainer}
    />
  </div>
</div>