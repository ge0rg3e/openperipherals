<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	type ListProps = TabsPrimitive.ListProps;

	let { class: className, children, ...restProps }: ListProps = $props();
</script>

<TabsPrimitive.List
	class={cn(
		"inline-flex w-fit items-center justify-center rounded-full border border-border bg-background p-1 text-muted-foreground",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</TabsPrimitive.List>
