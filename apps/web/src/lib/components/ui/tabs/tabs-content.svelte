<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	type ContentProps = TabsPrimitive.ContentProps;

	let { class: className, children, ...restProps }: ContentProps = $props();
</script>

<TabsPrimitive.Content
	class={cn(
		"flex-1 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</TabsPrimitive.Content>
