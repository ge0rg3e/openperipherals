<script lang="ts">
  // Custom title bar for the frameless window: drag anywhere empty to move,
  // double-click to maximise. Close hides to the tray like the native button.
  import {onMount} from "svelte";
  import {System, Window} from "@wailsio/runtime";
  import {Button} from "$lib/components/ui/button";
  import Icon from "./Icon.svelte";
  import {FanService} from "../../bindings/openperipherals";
  import {app, notify, rescan, type View} from "../lib/store.svelte";

  const windowsDesktop = System.IsWindows();
  const tabs: {view: View; label: string; icon: "home" | "fan" | "image" | "gear"}[] = [
    {view: "home", label: "Home", icon: "home"},
    {view: "cooling", label: "Cooling", icon: "fan"},
    {view: "wallpaper", label: "Wallpaper", icon: "image"},
    {view: "settings", label: "Settings", icon: "gear"},
  ];
  // Device pages live under Home.
  const section = $derived(app.view === "keyboard" || app.view === "mouse" ? "home" : app.view);

  let scanning = $state(false);
  async function scan() {
    scanning = true;
    await rescan();
    scanning = false;
  }

  // Fan control problems stay visible from every page.
  const fanNotice = $derived(app.fans?.notice ?? "");
  const needsAdmin = $derived(windowsDesktop && !!fanNotice && app.fans?.elevated === false);
  function fixFans() {
    if (needsAdmin) FanService.RestartAsAdmin().catch(notify);
    else app.view = "cooling";
  }

  let maximised = $state(false);
  const sync = () => Window.IsMaximised().then((m) => (maximised = m)).catch(() => {});
  onMount(() => {
    sync();
    window.addEventListener("resize", sync);
    return () => window.removeEventListener("resize", sync);
  });
</script>

<header class="col-span-full flex min-w-0 items-center gap-2 border-b border-border bg-[#090a0b] pl-3" style="--wails-draggable: drag">
  <div class="grid grid-cols-2 gap-0.5" aria-hidden="true">
    <span class="size-2 rounded-[2px] bg-primary"></span><span class="size-2 rounded-[2px] bg-foreground/70"></span>
    <span class="size-2 rounded-[2px] bg-foreground/70"></span><span class="size-2 rounded-[2px] bg-primary"></span>
  </div>
  <span class="mr-3 whitespace-nowrap text-[11px] font-semibold tracking-wide max-[680px]:mr-0">OpenPeripherals<span class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[8px] font-semibold uppercase tracking-wider text-muted-foreground">Beta</span></span>
  <nav class="flex h-full items-center gap-0.5" style="--wails-draggable: no-drag" aria-label="Main navigation">
    {#each tabs as item}
      <Button variant="ghost" class={section === item.view
        ? "inline-flex h-8 w-auto gap-1.5 rounded-md bg-muted px-2 text-foreground hover:bg-muted"
        : "inline-flex h-8 w-auto gap-1.5 rounded-md bg-transparent px-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"}
        aria-label={item.label} aria-current={section === item.view ? "page" : undefined} title={item.label}
        onclick={() => (app.view = item.view)}><Icon name={item.icon} size={14} /><span class="text-[10px] font-medium">{item.label}</span></Button>
    {/each}
  </nav>

  <div class="ml-auto flex h-full items-center gap-1 px-2" style="--wails-draggable: no-drag">
    {#if windowsDesktop && fanNotice}
      <Button variant="outline" class="inline-flex h-8 items-center gap-2 rounded-md border border-amber-500/25 bg-amber-500/10 px-2.5 text-xs text-amber-200 max-[680px]:[&_span]:hidden" title={needsAdmin ? `${fanNotice} Click to restart as administrator.` : fanNotice} onclick={fixFans}>
        <Icon name="shield" size={14} />
        <span>{needsAdmin ? "Fans need admin" : "Fan control unavailable"}</span>
      </Button>
    {/if}
    <Button variant="ghost" size="icon" class={app.sidebar
      ? "inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-muted text-foreground"
      : "inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-transparent text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"} title={app.sidebar ? "Hide side panel" : "Show side panel"}
            aria-label="Toggle side panel" onclick={() => (app.sidebar = !app.sidebar)}>
      <Icon name="panel" size={14} />
    </Button>
    <Button variant="ghost" size="icon" class={scanning
      ? "inline-flex size-7 animate-spin items-center justify-center rounded-md border border-transparent bg-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
      : "inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-transparent text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"} title="Rescan devices" aria-label="Rescan devices" onclick={scan} disabled={scanning}>
      <Icon name="refresh" size={14} />
    </Button>
  </div>

  <div class="flex h-full" style="--wails-draggable: no-drag">
    <Button variant="ghost" size="icon" class="grid h-full w-8 place-items-center rounded-none border-0 bg-transparent text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="Minimise" title="Minimise" onclick={() => Window.Minimise()}><Icon name="minimize" size={13} /></Button>
    <Button variant="ghost" size="icon" class="grid h-full w-8 place-items-center rounded-none border-0 bg-transparent text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label={maximised ? "Restore" : "Maximise"} title={maximised ? "Restore" : "Maximise"}
            onclick={() => Window.ToggleMaximise().then(sync)}>
      <Icon name={maximised ? "restore" : "maximize"} size={13} />
    </Button>
    <Button variant="ghost" size="icon" class="grid h-full w-8 place-items-center rounded-none border-0 bg-transparent text-muted-foreground transition-colors hover:bg-destructive hover:text-white" aria-label="Close to tray" title="Close to tray" onclick={() => Window.Close()}><Icon name="close" size={13} /></Button>
  </div>
</header>
