<script>
  import * as Popover from "$lib/components/ui/popover";
  import Input from "$lib/components/ui/input/input.svelte";
  import RangeCalendar from "$lib/components/ui/range-calendar/range-calendar.svelte";
  import { CalendarDate, getLocalTimeZone, today } from "@internationalized/date";
  import Button from "$lib/components/ui/button/button.svelte";
  import Icon from "@iconify/svelte";
  import { cn } from "$lib/utils";
  import { buttonVariants } from "$lib/components/ui/button";
  import { toast } from "svelte-sonner";


  /** @type {number}*/
  export let StartTimeRef;

  /** @type {number}*/
  export let EndTimeRef;

  /** @type {{start: CalendarDate, end: CalendarDate}}*/
  let logDayRange = {
    start: today(getLocalTimeZone()),
    end: today(getLocalTimeZone())
  };

  /** @type {{start: string, end: string}}*/
  let logTimeRange = {
    start: dateToTimeString(new Date(Date.now() - 1000 * 60 * 60)),
    end: dateToTimeString(new Date(Date.now())),
  }

  /**
   * Converts Date to a HH:MM string
   * @param {Date} date
   * @returns {string}
   */
  function dateToTimeString(date) {
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    return `${hours}:${minutes}`;
  }

  /** 
   * Adds a HH:MM string to the provided date and returns the unix time.
   * @param {Date} date
   * @param {string} timeString
   * @returns {number}
  */
  function timeStringToUnix(date, timeString) {
    const [hours, minutes] = timeString.split(':');
    return date.setHours(Number(hours), Number(minutes), 0, 0);
  }
</script>

<div class="flex flex-row gap-1 w-full lg:w-[300px]">
  <div class="flex flex-row items-center justify-center gap-2 p-1 w-full text-sm bg-black border-[1px] border-slate-200/15 rounded-lg">
    <span class="text-nowrap">{new Date(StartTimeRef).toLocaleTimeString("en-US", {
      month: "2-digit",
      day: "2-digit",
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
    })}</span>
    <span class="font-bold">-</span>
    <span class="text-nowrap">{new Date(EndTimeRef).toLocaleTimeString("en-US", {
      month: "2-digit",
      day: "2-digit",
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
    })}</span>
  </div>
  <Popover.Root>
    <Popover.Trigger class="{cn(buttonVariants({variant: "ghost"}))}"><Icon icon="uiw:date" /></Popover.Trigger>
    <Popover.Content class="flex flex-col gap-2 items-center w-max">
      <RangeCalendar bind:value={logDayRange} />
      <div class="flex flex-row gap-4 justify-center items-center">
        <Input bind:value={logTimeRange.start} type="time" placeholder="start time" />
        <span class="font-bold">-</span>
        <Input bind:value={logTimeRange.end} type="time" placeholder="end time" />
      </div>
      <Button class="w-full" type="submit" on:click={() => {
        if (!logDayRange.start || !logDayRange.end) {
          toast("Validation Error", {
            description: "You must specify a range"
          })
          return;
        }

        StartTimeRef = timeStringToUnix(
          logDayRange.start?.toDate(getLocalTimeZone()) ?? new Date(Date.now()), logTimeRange.start
        );
        EndTimeRef = timeStringToUnix(
          logDayRange.end?.toDate(getLocalTimeZone()) ?? new Date(Date.now()), logTimeRange.end
        );

        toast("Success", {
          description: "Log range updated"
        })
      }}>Apply Range</Button>
    </Popover.Content>
  </Popover.Root>
</div>