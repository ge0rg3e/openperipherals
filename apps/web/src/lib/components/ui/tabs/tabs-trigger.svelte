<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	type TriggerProps = TabsPrimitive.TriggerProps;

	let { class: className, children, ...restProps }: TriggerProps = $props();
</script>

<TabsPrimitive.Trigger
	class={cn(
		"inline-flex items-center justify-center rounded-full px-4 py-1.5 text-sm font-medium transition-colors whitespace-nowrap focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-foreground data-[state=active]:text-background",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</TabsPrimitive.Trigger>
