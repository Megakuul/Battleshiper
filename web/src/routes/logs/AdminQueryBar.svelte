<script>
  import * as Select from "$lib/components/ui/select";
  import Input from "$lib/components/ui/input/input.svelte";
  import { ProjectInfo } from "$lib/stores";
  import { toast } from "svelte-sonner";
  import LoaderCircle from "lucide-svelte/icons/loader-circle";
  import Button from "$lib/components/ui/button/button.svelte";
  import { FetchLog } from "$lib/adapter/admin/fetchlog";
  import DatePicker from "./DatePicker.svelte";

  const ADMIN_LOG_TYPES = [
    "api",
    "pipeline"
  ];

  /** @type {import("$lib/adapter/admin/fetchlog").fetchLogInput}*/
  export let CurrentLogInputRef;

  /** @type {import("$lib/adapter/admin/fetchlog").fetchLogOutput|undefined}*/
  export let CurrentLogOutputRef;

  /** @type {string} */
  export let ExceptionRef;

  /** @type {boolean}*/
  let queryButtonState;
</script>

<div class="flex flex-col w-10/12 bg-slate-700/20 rounded-lg">
  <div class="flex flex-col lg:flex-row gap-4 justify-between p-3">
    <Select.Root selected={{value: CurrentLogInputRef.log_type, label: CurrentLogInputRef.log_type}} onSelectedChange={(v) => {
      if (v && v.value) {
        CurrentLogInputRef.log_type = v.value;
      } else {
        toast("Validation Exception", {
          description: "Failed to apply logtype",
        })
      }
    }}>
      <Select.Trigger class="w-full lg:w-[280px]">
        <Select.Value placeholder="Log Type" />
      </Select.Trigger>
      <Select.Content>
        {#each ADMIN_LOG_TYPES as logtype}
          <Select.Item class="text-sm" value="{logtype}">{logtype}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>

    <Input bind:value={CurrentLogInputRef.count} type="number" placeholder="limit" class="w-full lg:w-[70px] mr-auto" />

    <DatePicker bind:StartTimeRef={CurrentLogInputRef.start_time} bind:EndTimeRef={CurrentLogInputRef.end_time} />
  </div>

  <Button class="m-4" variant="secondary" type="submit" on:click={async () => {
    try {
      queryButtonState = true;
      CurrentLogOutputRef = await FetchLog(CurrentLogInputRef);
      toast.success("Success", {
        description: CurrentLogOutputRef.message
      })
    } catch (/** @type {any} */ err) {
      ExceptionRef = err.message;
      toast.error("Exception", {
        description: "Failed to fetch logs",
      })
    }
    queryButtonState = false;
  }}>
    Query Logs
    {#if queryButtonState}
      <LoaderCircle class="ml-2 h-4 w-4 animate-spin" />
    {/if}
  </Button>
</div>