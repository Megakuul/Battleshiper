<script>
  /** @type {{timestamp: number, message: string}[]}*/
  export let LogEvents;

  /** 
   * @typedef {"INFO" | "WARN" | "ERROR"} LOG_LEVEL
  */

  /** 
   * Analyzes the message and returns a message level, based on the message content. 
   * @param {string} message
   * @returns {LOG_LEVEL}
  */
  function getMessageLevel(message) {
    const lowercaseMessage = message.toLowerCase();
    if (lowercaseMessage.includes("error") || lowercaseMessage.includes("fail")) {
      return "ERROR";
    }
    if (lowercaseMessage.includes("warn")) {
      return "WARN";
    }
    return "INFO";
  }
</script>

<div class="flex flex-col gap-2 w-10/12 p-4 h-[60vh] bg-slate-700/20 rounded-lg overflow-scroll-hidden text-xs sm:text-base">
  {#each LogEvents.reverse() as event}
    {@const level = getMessageLevel(event.message)}
    <div class="flex flex-row justify-start items-start gap-2">
      <p class="text-slate-200/70 font-bold text-nowrap">{new Date(event.timestamp).toLocaleDateString("en-US", {
        month: "2-digit",
        day: "2-digit",
        hour12: false,
        hour: "2-digit",
        minute: "2-digit",
      })}</p>
      <p class="font-bold text-nowrap" class:info={level === "INFO"} class:warn={level === "WARN"} class:error={level === "ERROR"}>
        [{level}]<span class="text-slate-200/70">:</span>
      </p>
      <p class="break-all">{event.message}</p>
    </div>
  {/each}
</div>

<style>
  .info {
    @apply text-slate-200/70;
  }
  .warn {
    @apply text-orange-600/80;
  }
  .error {
    @apply text-red-600/80;
  }
</style>
