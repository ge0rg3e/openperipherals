<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	type RootProps = TabsPrimitive.RootProps;

	let {
		class: className,
		children,
		value = $bindable(),
		...restProps
	}: RootProps = $props();
</script>

<TabsPrimitive.Root
	bind:value
	class={cn("flex flex-col", className)}
	{...restProps}
>
	{@render children?.()}
</TabsPrimitive.Root>
