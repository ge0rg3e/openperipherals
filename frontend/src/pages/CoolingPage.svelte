<script lang="ts">
  import {Badge} from "$lib/components/ui/badge";
  import {Button} from "$lib/components/ui/button";
  import {Input} from "$lib/components/ui/input";
  import PageHeader from "../ui/PageHeader.svelte";
  import FanCard from "../FanCard.svelte";
  import { app, notify, refreshFans } from "../lib/store.svelte";
  import {
    FanService,
    PeripheralService,
    type FanConfig,
    type FanPreset,
  } from "../../bindings/openperipherals";

  // Bumped when a preset is applied so the cards recreate their editors from
  // the new configs instead of keeping the old local state.
  let epoch = $state(0);
  let busy = $state("");
  let fixing = $state(false);
  let creating = $state(false);
  let draftName = $state("");
  let saving = $state(false);

  // The preset the user last applied. It stays selected while fan settings are
  // edited, so the card can offer Update.
  let appliedId = $state("");

  const fans = $derived(app.fans?.fans ?? []);
  const presets = $derived(app.fans?.presets ?? []);

  // Enabling startup also relaunches this unelevated copy once, so Windows
  // shows a single UAC prompt and the app then starts elevated from the task.
  async function alwaysAdmin() {
    if (fixing) return;
    fixing = true;
    try {
      await PeripheralService.SetAutostart(true);
    } catch (e) {
      notify(e);
      fixing = false;
    }
  }

  const samePoints = (a: number[][] | null, b: number[][] | null) =>
    (a?.length ?? 0) === (b?.length ?? 0) &&
    (a ?? []).every((p, i) => p[0] === b![i][0] && p[1] === b![i][1]);

  /** Compare only the fields a mode actually uses. */
  function sameConfig(a: FanConfig, b: FanConfig) {
    if (a.mode !== b.mode) return false;
    if (a.mode === "fixed") return a.fixed === b.fixed;
    if (a.mode === "curve")
      return a.sensor === b.sensor && samePoints(a.points, b.points);
    return true;
  }

  function matches(p: FanPreset) {
    const entries = p.configs ?? [];
    if (entries.length !== fans.length) return false;
    return fans.every((fan) => {
      const e = entries.find((entry) => entry.fanId === fan.id);
      return !!e && sameConfig(e.config, fan.config);
    });
  }

  const matchedId = $derived.by(
    () => presets.find((p) => matches(p))?.id ?? "",
  );
  const currentId = $derived(
    presets.some((p) => p.id === appliedId) ? appliedId : matchedId,
  );
  const current = $derived(presets.find((p) => p.id === currentId) ?? null);
  const dirty = $derived(!!current && !matches(current));

  // Adopt an exact match on load, and forget a preset that was deleted.
  $effect(() => {
    if (!appliedId && matchedId) appliedId = matchedId;
    else if (appliedId && !presets.some((p) => p.id === appliedId))
      appliedId = "";
  });

  function summary(p: FanPreset) {
    if (p.builtin) return "Every fan follows firmware";
    const configs = (p.configs ?? []).map((e) => e.config);
    if (!configs.length) return "No fans";
    const curves = configs.filter((c) => c.mode === "curve").length;
    const fixed = configs.filter((c) => c.mode === "fixed");
    const auto = configs.filter((c) => c.mode === "auto").length;
    const parts: string[] = [];
    if (curves)
      parts.push(curves === configs.length ? "Curves" : `${curves} curves`);
    if (fixed.length) {
      const speeds = [...new Set(fixed.map((c) => c.fixed))];
      parts.push(speeds.length === 1 ? `Fixed ${speeds[0]}%` : "Fixed speeds");
    }
    if (auto) parts.push(auto === configs.length ? "Auto" : `${auto} auto`);
    return `${configs.length === 1 ? "1 fan" : `${configs.length} fans`} · ${parts.join(" · ")}`;
  }

  async function apply(p: FanPreset) {
    if (busy) return;
    busy = p.id;
    try {
      await FanService.ApplyPreset(p.id);
      appliedId = p.id;
      await refreshFans();
      epoch++;
    } catch (e) {
      notify(e);
    }
    busy = "";
  }

  function startNew() {
    creating = true;
    draftName = "";
  }

  function cancelNew() {
    creating = false;
    draftName = "";
  }

  async function createPreset(e: SubmitEvent) {
    e.preventDefault();
    const name = draftName.trim();
    if (!name || saving) return;
    saving = true;
    try {
      appliedId = await FanService.SavePreset(name);
      await refreshFans();
      creating = false;
      draftName = "";
    } catch (err) {
      notify(err);
    }
    saving = false;
  }

  async function updatePreset(p: FanPreset) {
    if (busy || p.builtin) return;
    busy = p.id;
    try {
      await FanService.UpdatePreset(p.id);
      appliedId = p.id;
      await refreshFans();
    } catch (e) {
      notify(e);
    }
    busy = "";
  }

  async function remove(p: FanPreset) {
    if (busy || !window.confirm(`Delete the "${p.name}" preset?`)) return;
    busy = p.id;
    try {
      await FanService.DeletePreset(p.id);
      if (appliedId === p.id) appliedId = "";
      await refreshFans();
    } catch (e) {
      notify(e);
    }
    busy = "";
  }
</script>

<PageHeader title="Cooling" />

{#if app.fans}
  {#if app.fans.notice}
    <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-500/25 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
      <span>{app.fans.notice}</span>
      {#if !app.fans.elevated}
        <span class="flex flex-wrap gap-2">
          <Button variant="outline" onclick={() => FanService.RestartAsAdmin().catch(notify)}
            >Restart as administrator</Button
          >
          <Button
            variant="ghost"
            onclick={alwaysAdmin}
            disabled={fixing}
            title="Start OpenPeripherals as administrator with Windows, so cooling works without asking again"
          >
            {fixing ? "Enabling…" : "Always run as admin"}
          </Button>
        </span>
      {/if}
    </div>
  {/if}

  {#if fans.length}
    <section class="settings-section preset-panel">
      <div class="mb-2 flex items-center justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold tracking-tight">Presets</h3>

        </div>
        {#if !creating}
          <Button variant="outline" onclick={startNew}>+ New preset</Button>
        {/if}
      </div>

      {#if creating}
        <form class="mb-4 flex flex-wrap gap-2" onsubmit={createPreset}>
          <Input
            bind:value={draftName}
            type="text"
            maxlength={32}
            autofocus
            aria-label="Preset name"
            placeholder="Preset name, e.g. Gaming or Silent"
          />
          <Button
            variant="default"
            type="submit"
            disabled={!draftName.trim() || saving}
          >
            {saving ? "Saving…" : "Save current settings"}
          </Button>
          <Button
            variant="outline"
            type="button"
            onclick={cancelNew}
            disabled={saving}>Cancel</Button
          >
        </form>
      {/if}

      {#if presets.length}
        <div class="preset-grid">
          {#each presets as p (p.id)}
            <div
              class={p.id === currentId
                ? "flex min-w-0 items-center justify-between gap-2 border-l-2 border-primary bg-primary/5 px-2.5 py-2 transition-colors hover:bg-primary/10 focus-visible:outline-2 focus-visible:outline-ring"
                : "flex min-w-0 items-center justify-between gap-2 border-l-2 border-transparent px-2.5 py-2 transition-colors hover:bg-muted/40 focus-visible:outline-2 focus-visible:outline-ring"}
              role="button"
              tabindex="0"
              title={`Apply ${p.name}`}
              aria-label={`Apply ${p.name}`}
              onclick={() => apply(p)}
              onkeydown={(e) => {
                if (e.target !== e.currentTarget) return;
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  apply(p);
                }
              }}
            >
              <div class="grid min-w-0 gap-1">
                <span class="truncate text-xs font-semibold">
                  {p.name}
                  {#if p.id === currentId && dirty}
                    <Badge variant="secondary" class="ml-2 rounded-full bg-amber-500/15 px-1.5 py-0.5 text-[9px] text-amber-200">Changed</Badge>
                  {/if}
                </span>
                <span class="truncate text-[10px] text-muted-foreground">{summary(p)}</span>
              </div>
              <div class="flex shrink-0 items-center gap-1">
                {#if p.id === currentId && dirty && !p.builtin}
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={(e) => {
                      e.stopPropagation();
                      updatePreset(p);
                    }}
                    disabled={!!busy}>Update</Button
                  >
                {/if}
                {#if !p.builtin}
                  <Button
                    variant="destructive"
                    size="icon-xs"
                    title="Delete preset"
                    aria-label={`Delete ${p.name}`}
                    onclick={(e) => {
                      e.stopPropagation();
                      remove(p);
                    }}
                    disabled={!!busy}>✕</Button
                  >
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {:else if !creating}
        <div class="flex flex-wrap items-center justify-between gap-4 py-3">
          <div>
            <b>No presets yet</b>
            <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
              Set the fan cards below, then save those values here to switch to
              them later.
            </p>
          </div>
          <Button onclick={startNew}>Create preset</Button>
        </div>
      {/if}
    </section>
  {/if}

  {#key epoch}
    <div class="fan-grid">
      {#each fans as fan (fan.id)}
        <FanCard {fan} sensors={app.fans.sensors ?? []} onerror={notify} />
      {:else}
        <p class="text-xs leading-relaxed text-muted-foreground">No controllable fans found.</p>
      {/each}
    </div>
  {/key}
{:else}
  <p class="text-xs leading-relaxed text-muted-foreground">Detecting fans…</p>
{/if}
