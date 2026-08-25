import keymap_100 from './keymap_100_default.json';
import keymap_80 from './keymap_80_default.json';
import keymap_65 from './keymap_65_default.json';
import keymap_60 from './keymap_60_default.json';

// Key codes per physical layout position, keyed by the app's LayoutKind.
const KEYMAPS = {
	100: keymap_100,
	80: keymap_80,
	65: keymap_65,
	60: keymap_60
};

export default KEYMAPS;
