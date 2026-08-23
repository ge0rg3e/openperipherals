import layout_100 from './layout_100_default.json';
import layout_80 from './layout_80_default.json';
import layout_65 from './layout_65_default.json';
import layout_60 from './layout_60_default.json';

// Physical key geometry (KLE units) per board size, keyed by the app's LayoutKind.
const LAYOUTS = {
	100: layout_100,
	80: layout_80,
	65: layout_65,
	60: layout_60
};

export default LAYOUTS;
