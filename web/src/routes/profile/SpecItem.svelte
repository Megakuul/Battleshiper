<script>
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  import * as Avatar from "$lib/components/ui/avatar";
  import { cn } from "$lib/utils";

  /** @type {string} */
  export let title;

  /** @type {any} */
  export let value;

  /** @type {string} */
  export let description;

  /** @type {any} */
  export let className = undefined;
  export { className as class };
</script>


<div class="{cn("flex flex-col w-48 items-center", className)}" {...$$restProps}>
  <Tooltip.Root>
    <Tooltip.Trigger>
      <Avatar.Root class="h-20 w-20 shadow-lg">
        <Avatar.Fallback class="bg-slate-800">
          <span class="font-bold animate-pulse text-xl">{value}</span>
        </Avatar.Fallback>
      </Avatar.Root>
    </Tooltip.Trigger>
    <Tooltip.Content>
      <p>{description}</p>
    </Tooltip.Content>
  </Tooltip.Root>
  <h2 class="text-lg opacity-80 font-bold text-center mt-2">{title}</h2>
</div>