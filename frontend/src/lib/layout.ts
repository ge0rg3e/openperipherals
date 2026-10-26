// Physical layout of the Razer Huntsman Elite in key units (1u = one
// alphanumeric key), which other boards are cut from (layoutFor). Key ids
// are KeyboardEvent.code values so key presses can be matched to caps
// directly.

export interface Cap {
  id: string;
  label: string;
  x: number;
  y: number;
  w: number;
  h: number;
  round?: boolean; // media buttons
}

export const WIDTH = 22.5;
export const HEIGHT = 6.5;

/** The multi-function dial, top right (centre and radius). */
export const DIAL = {x: 22.0, y: 0.5, r: 0.5};

type Item = [id: string, label: string, w?: number, h?: number] | number; // number = gap

function row(y: number, items: Item[], out: Cap[]) {
  let x = 0;
  for (const it of items) {
    if (typeof it === "number") {
      x += it;
      continue;
    }
    const [id, label, w = 1, h = 1] = it;
    out.push({id, label, x, y, w, h});
    x += w;
  }
}

const letters = (s: string): Item[] => [...s].map((c) => [`Key${c}`, c] as Item);
const digits = (s: string): Item[] => [...s].map((c) => [`Digit${c}`, c] as Item);
const fkeys = (from: number, to: number): Item[] =>
  Array.from({length: to - from + 1}, (_, i) => [`F${from + i}`, `F${from + i}`] as Item);

export const CAPS: Cap[] = (() => {
  const k: Cap[] = [];
  row(0, [["Escape", "Esc"], 1, ...fkeys(1, 4), 0.5, ...fkeys(5, 8), 0.5, ...fkeys(9, 12), 0.25,
    ["PrintScreen", "PrtSc"], ["ScrollLock", "ScrLk"], ["Pause", "Pause"]], k);
  row(1.5, [["Backquote", "`"], ...digits("1234567890"), ["Minus", "-"], ["Equal", "="], ["Backspace", "Back", 2], 0.25,
    ["Insert", "Ins"], ["Home", "Home"], ["PageUp", "PgUp"], 0.25,
    ["NumLock", "Num"], ["NumpadDivide", "/"], ["NumpadMultiply", "*"], ["NumpadSubtract", "-"]], k);
  row(2.5, [["Tab", "Tab", 1.5], ...letters("QWERTYUIOP"), ["BracketLeft", "["], ["BracketRight", "]"], ["Backslash", "\\", 1.5], 0.25,
    ["Delete", "Del"], ["End", "End"], ["PageDown", "PgDn"], 0.25,
    ["Numpad7", "7"], ["Numpad8", "8"], ["Numpad9", "9"], ["NumpadAdd", "+", 1, 2]], k);
  row(3.5, [["CapsLock", "Caps", 1.75], ...letters("ASDFGHJKL"), ["Semicolon", ";"], ["Quote", "'"], ["Enter", "Enter", 2.25], 3.5,
    ["Numpad4", "4"], ["Numpad5", "5"], ["Numpad6", "6"]], k);
  row(4.5, [["ShiftLeft", "Shift", 2.25], ...letters("ZXCVBNM"), ["Comma", ","], ["Period", "."], ["Slash", "/"], ["ShiftRight", "Shift", 2.75], 1.25,
    ["ArrowUp", "↑"], 1.25, ["Numpad1", "1"], ["Numpad2", "2"], ["Numpad3", "3"], ["NumpadEnter", "Ent", 1, 2]], k);
  row(5.5, [["ControlLeft", "Ctrl", 1.25], ["MetaLeft", "Win", 1.25], ["AltLeft", "Alt", 1.25], ["Space", "", 6.25],
    ["AltRight", "Alt", 1.25], ["Fn", "Fn", 1.25], ["ContextMenu", "Menu", 1.25], ["ControlRight", "Ctrl", 1.25], 0.25,
    ["ArrowLeft", "←"], ["ArrowDown", "↓"], ["ArrowRight", "→"], 0.25, ["Numpad0", "0", 2], ["NumpadDecimal", "."]], k);
  // Media buttons above the numpad, left of the dial.
  const media: [string, string][] = [["MediaTrackPrevious", "⏮"], ["MediaPlayPause", "⏯"], ["MediaTrackNext", "⏭"]];
  media.forEach(([id, label], i) => k.push({id, label, x: 18.6 + i * 0.85, y: 0.15, w: 0.7, h: 0.7, round: true}));
  return k;
})();

export interface Layout {
  caps: Cap[];
  width: number;
  height: number;
  dial: boolean;
}

const layouts = new Map<string, Layout>();

/**
 * The caps of a board size, cut from the full layout: "tkl" drops the
 * numpad, "mini" (60%) also the function row, nav cluster and arrows.
 * `media` keeps the Huntsman Elite's media keys and dial.
 */
export function layoutFor(kind = "full", media = false): Layout {
  const key = `${kind}:${media}`;
  let l = layouts.get(key);
  if (!l) {
    let caps = CAPS.filter((c) => media || !c.round);
    let width = WIDTH;
    let height = HEIGHT;
    if (kind === "tkl") {
      caps = caps.filter((c) => c.x < 18.5);
      width = 18.25;
    } else if (kind === "mini") {
      caps = caps.filter((c) => c.x < 15 && c.y >= 1.5).map((c) => ({...c, y: c.y - 1.5}));
      width = 15;
      height = 5;
    }
    l = {caps, width, height, dial: media && kind === "full"};
    layouts.set(key, l);
  }
  return l;
}

// Windows virtual-key codes for each cap, as the key blocker expects them.
// EXT / PLAIN narrow a code to the extended or non-extended variant, so e.g.
// numpad Enter and Enter can be disabled separately.
const EXT = 0x100;
const PLAIN = 0x200;

const SPECIAL: Record<string, number> = {
  Escape: 0x1b, Backquote: 0xc0, Minus: 0xbd, Equal: 0xbb, Backspace: 0x08, Tab: 0x09,
  BracketLeft: 0xdb, BracketRight: 0xdd, Backslash: 0xdc, CapsLock: 0x14, Semicolon: 0xba, Quote: 0xde,
  Enter: 0x0d | PLAIN, ShiftLeft: 0xa0, ShiftRight: 0xa1, Comma: 0xbc, Period: 0xbe, Slash: 0xbf,
  ControlLeft: 0xa2, ControlRight: 0xa3, MetaLeft: 0x5b, AltLeft: 0xa4, AltRight: 0xa5, ContextMenu: 0x5d, Space: 0x20,
  PrintScreen: 0x2c, ScrollLock: 0x91, Pause: 0x13,
  Insert: 0x2d | EXT, Home: 0x24 | EXT, PageUp: 0x21 | EXT, Delete: 0x2e | EXT, End: 0x23 | EXT, PageDown: 0x22 | EXT,
  ArrowLeft: 0x25 | EXT, ArrowUp: 0x26 | EXT, ArrowRight: 0x27 | EXT, ArrowDown: 0x28 | EXT,
  NumLock: 0x90, NumpadDivide: 0x6f, NumpadMultiply: 0x6a, NumpadSubtract: 0x6d, NumpadAdd: 0x6b,
  NumpadDecimal: 0x6e, NumpadEnter: 0x0d | EXT,
  MediaTrackPrevious: 0xb1, MediaPlayPause: 0xb3, MediaTrackNext: 0xb0,
};

/** The dial's three actions, each disabled separately. */
export const DIAL_ACTIONS = [
  {id: "dial:volume_up", label: "Turn right", detail: "Volume up", code: 0xaf},
  {id: "dial:mute", label: "Press", detail: "Mute", code: 0xad},
  {id: "dial:volume_down", label: "Turn left", detail: "Volume down", code: 0xae},
] as const;

/** Virtual-key code for a cap, or null if Windows never sees it (Fn). */
export function keyCode(id: string): number | null {
  let m: RegExpMatchArray | null;
  if ((m = id.match(/^Key([A-Z])$/))) return m[1].charCodeAt(0);
  if ((m = id.match(/^Digit(\d)$/))) return 0x30 + +m[1];
  if ((m = id.match(/^Numpad(\d)$/))) return 0x60 + +m[1];
  if ((m = id.match(/^F(\d+)$/))) return 0x6f + +m[1];
  return SPECIAL[id] ?? null;
}

/** True for keys whose presses can be traced to the keyboard (media keys). */
export const isMediaKey = (id: string) => id.startsWith("dial:") || id.startsWith("Media");
