// App-wide state shared by the tiles, device pages, presets and sensors.
import {Events} from "@wailsio/runtime";
import {
  FanService,
  PeripheralService,
  SystemService,
  type DPIStage,
  type FanStatus,
  type KeyboardLighting,
  type Status,
  type SyncLighting,
} from "../../bindings/openperipherals";
import type {Stats} from "../../bindings/openperipherals/internal/sysstats";

export type View = "home" | "keyboard" | "mouse" | "cooling" | "settings";

export const DPI_ZONE = 2;
const DEFAULT_MOUSE_LIGHTING: KeyboardLighting = {
  effect: "static", color: "#33e19a", color2: "#1f7bff", mode: "single", speed: 2, direction: "right",
};
const HISTORY = 60;

export const app = $state({
  view: "home" as View,
  sidebar: true,
  status: null as Status | null,
  fans: null as FanStatus | null,
  stats: null as Stats | null,
  kb: null as {lighting: KeyboardLighting; brightness: number} | null,
  /** Editable mouse state. The SABRE uses colors/stages/current/brightness;
   *  Razer and Pulsar mice use dpi, and Razer mice lighting. */
  mouse: null as {
    colors: string[];
    stages: DPIStage[];
    current: number;
    brightness: number;
    dpi: number;
    lighting: KeyboardLighting;
  } | null,
  history: {} as Record<string, number[]>,
  /** Disabled keys: id -> virtual-key code. */
  disabled: {} as Record<string, number>,
  /** One effect and colour applied to every device. */
  sync: {enabled: false, effect: "static", color: "#33e19a"} as SyncLighting,
  error: "",
});

let errorTimer: ReturnType<typeof setTimeout>;
export function notify(e: unknown) {
  app.error = e instanceof Error ? e.message : String(e);
  clearTimeout(errorTimer);
  errorTimer = setTimeout(() => (app.error = ""), 5000);
}

function record(id: string, v: number) {
  const h = app.history[id] ?? [];
  h.push(v);
  if (h.length > HISTORY) h.shift();
  app.history[id] = h;
}

/** Adopt a device snapshot; editable state is seeded when a device appears. */
function setStatus(s: Status) {
  app.status = s;
  if (!s.keyboard) app.kb = null;
  else if (!app.kb) app.kb = {lighting: {...s.keyboard.lighting}, brightness: s.keyboard.brightness};
  if (!s.mouse) app.mouse = null;
  else if (!app.mouse)
    app.mouse = {
      colors: [...(s.mouse.colors ?? [])],
      stages: (s.mouse.stages ?? []).map((st) => ({...st})),
      current: s.mouse.current,
      brightness: s.mouse.brightness,
      dpi: s.mouse.dpi,
      lighting: {...(s.mouse.lighting ?? DEFAULT_MOUSE_LIGHTING)},
    };
}

function setFans(f: FanStatus) {
  app.fans = f;
  for (const s of f.sensors ?? []) if (s.temp >= 0) record(`t:${s.id}`, s.temp);
  for (const fan of f.fans ?? []) {
    if (fan.rpm >= 0) record(`rpm:${fan.id}`, fan.rpm);
    if (fan.duty >= 0) record(`duty:${fan.id}`, fan.duty);
  }
}

async function sample() {
  if (document.visibilityState !== "visible") return;
  try {
    const s = await SystemService.Stats();
    app.stats = s;
    if (s.cpu >= 0) record("cpu", s.cpu);
    record("ram", s.ramUsed / 1024 ** 3);
    s.gpus?.forEach((g, i) => {
      record(`gpu${i}`, g.usage);
      record(`vram${i}`, g.vramUsed / 1024 ** 3);
    });
  } catch {
    /* next tick retries */
  }
}

/** Load everything and subscribe to live updates. Returns a cleanup. */
export function start(): () => void {
  PeripheralService.Status().then(setStatus).catch(notify);
  PeripheralService.DisabledKeys().then((d) => (app.disabled = (d ?? {}) as Record<string, number>)).catch(notify);
  PeripheralService.Sync().then((s) => (app.sync = s)).catch(notify);
  FanService.Status().then(setFans).catch(notify);
  const offStatus = Events.On("status", (e) => setStatus(e.data));
  const offFans = Events.On("fans", (e) => setFans(e.data));
  sample();
  const timer = setInterval(sample, 1000);
  document.addEventListener("visibilitychange", sample);
  return () => {
    offStatus();
    offFans();
    clearInterval(timer);
    document.removeEventListener("visibilitychange", sample);
  };
}

export async function rescan() {
  try {
    app.kb = null;
    app.mouse = null;
    setStatus(await PeripheralService.Refresh());
  } catch (e) {
    notify(e);
  }
}

// Keyboard

export const EFFECT_NAMES: Record<string, string> = {
  static: "Static", spectrum: "Spectrum", wave: "Wave", wheel: "Wheel", breathing: "Breathing",
  reactive: "Reactive", starlight: "Starlight", off: "Off",
};

/** Effects that use the colour pickers (unless their mode is random). */
export const COLOR_EFFECTS = ["static", "reactive", "breathing", "starlight"];

export function applyKeyboard() {
  if (!app.kb) return;
  endSync();
  const l = app.kb.lighting;
  const max = app.status?.keyboard?.speeds?.[l.effect] ?? 0;
  if (max) l.speed = Math.min(Math.max(l.speed || 1, 1), max);
  if (!app.status?.keyboard?.modes) l.mode = "single";
  PeripheralService.SetKeyboardLighting($state.snapshot(l)).catch(notify);
}

/** Razer keyboards: "gameMode" or "macroLeds", stored on the keyboard. */
export async function setKeyboardOption(option: string, on: boolean) {
  try {
    await PeripheralService.SetKeyboardOption(option, on);
  } catch (e) {
    notify(e);
  }
  refreshStatus();
}

export function applyBrightness() {
  if (app.kb) PeripheralService.SetKeyboardBrightness(app.kb.brightness).catch(notify);
}

// Key blocking

export async function setKeyDisabled(id: string, code: number, disabled: boolean) {
  try {
    await PeripheralService.SetKeyDisabled(id, code, disabled);
    if (disabled) app.disabled[id] = code;
    else delete app.disabled[id];
  } catch (e) {
    notify(e);
  }
}

// Sync

export const SYNC_EFFECTS = [
  ["static", "Static"], ["spectrum", "Spectrum"], ["wave", "Wave"], ["breathing", "Breathing"], ["off", "Off"],
] as const;

export const SYNC_COLORS = ["#33e19a", "#1f7bff", "#ff2a3d", "#ffaa00", "#8b4dff", "#00d5ff", "#e8f4ff"];

const syncAnimated = () => app.sync.enabled && ["spectrum", "wave", "breathing"].includes(app.sync.effect);

/** Apply the sync setting to every device (turning sync on unless told otherwise). */
export async function applySync(change: Partial<SyncLighting> = {}) {
  const next = {...$state.snapshot(app.sync), enabled: true, ...change};
  app.sync = next;
  // Mirror the result locally so previews match straight away.
  if (next.enabled) {
    if (app.kb) Object.assign(app.kb.lighting, {effect: next.effect, color: next.color, mode: "single", direction: "right"});
    if (app.mouse && !syncAnimated()) {
      const c = next.effect === "off" ? "#000000" : next.color;
      app.mouse.colors = app.mouse.colors.map((old, i) => (i === DPI_ZONE ? old : c));
    }
    if (app.mouse && app.status?.mouse?.effects?.length) {
      const effect = next.effect === "wave" ? "spectrum" : next.effect;
      Object.assign(app.mouse.lighting, {effect, color: next.color, color2: next.color, mode: "single"});
    }
  }
  sendSync(next);
}

// Sync changes go out one at a time and only the newest pending one is
// sent, so rapid clicks can't land on the devices out of order.
let syncInFlight = false;
let syncQueued: SyncLighting | null = null;
async function sendSync(next: SyncLighting) {
  syncQueued = next;
  if (syncInFlight) return;
  syncInFlight = true;
  while (syncQueued) {
    const s = syncQueued;
    syncQueued = null;
    try {
      await PeripheralService.SetSync(s);
    } catch (e) {
      notify(e);
    }
  }
  syncInFlight = false;
}

/** Direct edits on a device page end sync (the backend does the same). */
function endSync() {
  if (app.sync.enabled) app.sync.enabled = false;
}

// Mouse

/** Zone colours as shown on the mouse: the DPI LED follows the active stage. */
export function mouseZones(): string[] {
  if (!app.mouse) return [];
  const c = [...app.mouse.colors];
  c[DPI_ZONE] = app.mouse.stages[app.mouse.current]?.color ?? c[DPI_ZONE];
  return c;
}

/** Zone colours for drawing a Razer or Pulsar mouse: its underglow effect
 *  colour where it has lighting, unlit otherwise. */
export function mouseArtColors(): string[] {
  const l = app.mouse?.lighting;
  const lit = !!app.status?.mouse?.effects?.length && l && COLOR_EFFECTS.includes(l.effect) && l.mode !== "random";
  const c = lit ? l.color : "#1a1d22";
  return [c, c, "#1a1d22", c];
}

/** The effect a Razer mouse runs, in the form the mouse art animates. */
export function mouseArtSync(): SyncLighting | null {
  const l = app.mouse?.lighting;
  if (!l || !app.status?.mouse?.effects?.length) return null;
  return {enabled: true, effect: l.effect, color: l.color};
}

let previewTimer: ReturnType<typeof setTimeout> | undefined;
let previewPending = false;
/** Show colours live while a picker is open, at most every 40 ms. */
export function previewMouse() {
  endSync();
  previewPending = true;
  if (previewTimer) return;
  previewTimer = setTimeout(() => {
    previewTimer = undefined;
    if (!previewPending) return;
    previewPending = false;
    PeripheralService.PreviewMouseColors(mouseZones()).catch(notify);
  }, 40);
}

export function saveMouseColors() {
  endSync();
  previewPending = false;
  PeripheralService.SetMouseColors(mouseZones()).catch(notify);
}

export function applyMouseBrightness() {
  if (app.mouse) PeripheralService.SetMouseBrightness(app.mouse.brightness).catch(notify);
}

let dpiTimer: ReturnType<typeof setTimeout> | undefined;
export function applyDpi(delay = 300) {
  clearTimeout(dpiTimer);
  dpiTimer = setTimeout(() => {
    if (app.mouse) PeripheralService.SetMouseDPI($state.snapshot(app.mouse.stages), app.mouse.current).catch(notify);
  }, delay);
}

/** Re-read the device snapshot (e.g. after a write the device confirmed). */
async function refreshStatus() {
  try {
    setStatus(await PeripheralService.Status());
  } catch (e) {
    notify(e);
  }
}

// Razer and Pulsar mice: settings are written one at a time and confirmed by
// the mouse, then the snapshot is re-read so the page shows what it kept.

export async function setMouseOption(option: string, value: number) {
  try {
    await PeripheralService.SetMouseOption(option, value);
  } catch (e) {
    notify(e);
  }
  await refreshStatus();
  if (option === "dpi" && app.mouse && app.status?.mouse) app.mouse.dpi = app.status.mouse.dpi;
}

let sensorDpiTimer: ReturnType<typeof setTimeout> | undefined;
export function applySensorDpi(delay = 300) {
  clearTimeout(sensorDpiTimer);
  sensorDpiTimer = setTimeout(() => app.mouse && setMouseOption("dpi", app.mouse.dpi), delay);
}

export function applyMouseLighting() {
  if (!app.mouse) return;
  endSync();
  const l = app.mouse.lighting;
  if (l.effect === "reactive") l.speed = Math.min(Math.max(l.speed || 1, 1), 4);
  PeripheralService.SetMouseLighting($state.snapshot(l)).catch(notify);
}
