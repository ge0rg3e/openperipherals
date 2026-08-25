export type MatrixStyle = 'classic' | 'extended';

export type LayoutKind = 'full' | 'tkl' | 'mini' | 'small';

export interface KeyboardDevice {
	pid: number;
	name: string;
	/** USB HID interface index used in the control request. */
	reportIndex: number;
	/** transaction id used for command reports */
	transactionId: number;
	/** extended matrix (class 0x0f) vs classic (class 0x03) */
	style: MatrixStyle;
	/** extended-matrix wave "left" byte. OpenRazer: WAVE_RIGHT is always 1,
	 *  but "left" is 0 on old firmware and 2 on modern boards. Classic boards
	 *  always use 2 (WAVE_DIRS (1,2)). */
	waveLeft?: number;
	/** URL (from /static) of a product photo shown on the landing page. */
	image?: string;
	/** true when the board exposes a battery (wireless models + wired HyperSpeed dongles). */
	battery?: boolean;
	/** transaction id used for battery queries (differs from the lighting tid on some boards). */
	batteryTid?: number;
	layout: LayoutKind;
	/** LED matrix rows (canonical Chroma grid is 6×22). */
	matrixRows: number;
	/** LED matrix columns (canonical Chroma grid is 6×22). */
	matrixCols: number;
	/** false for boards whose driver has no custom-frame device file. */
	custom: boolean;
	/** true when the firmware exposes the hardware colour-wheel effect
	 *  (extended matrix effect 0x0a - only the BlackWidow V4 family). */
	wheel?: boolean;
}

interface RawSpec {
	pid: number;
	name: string;
	report: 1 | 2 | 3;
	tid: number;
	style: MatrixStyle;
	layout?: LayoutKind;
	waveLeft?: number;
	image?: string;
	battery?: boolean;
	batteryTid?: number;
	/** false for boards whose kernel driver exposes no matrix_custom_frame. */
	custom?: boolean;
	/** true when the board supports the hardware colour-wheel effect
	 *  (matrix_effect_wheel device file: BlackWidow V4 family). */
	wheel?: boolean;
}

/** LED matrix geometry (rows, cols). Boilerplate from the OpenRazer daemon
 *  (MATRIX_DIMS); only entries that differ from the layout-based default are
 *  listed here. */
const MATRIX_OVERRIDES: Record<number, [number, number]> = {
	// classic 6×16 laptop matrix (Blade family)
	0x020f: [6, 16],
	0x0220: [6, 16],
	0x0224: [6, 16],
	0x022d: [6, 16],
	0x0232: [6, 16],
	0x0233: [6, 16],
	0x0234: [6, 16],
	0x0239: [6, 16],
	0x023a: [6, 16],
	0x023b: [6, 16],
	0x0240: [6, 16],
	0x0245: [6, 16],
	0x0246: [6, 16],
	0x024a: [6, 16],
	0x024b: [6, 16],
	0x024c: [6, 16],
	0x024d: [6, 16],
	0x0252: [6, 16],
	0x0253: [6, 16],
	0x0255: [6, 16],
	0x0256: [6, 16],
	0x0259: [6, 16],
	0x0268: [6, 16],
	0x026a: [6, 16],
	0x026d: [6, 16],
	0x026e: [6, 16],
	0x026f: [6, 16],
	0x0270: [6, 16],
	0x0276: [6, 16],
	0x0279: [6, 16],
	0x027a: [6, 16],
	0x028a: [6, 16],
	0x028b: [6, 16],
	0x028c: [6, 16],
	0x029d: [6, 16],
	0x029e: [6, 16],
	0x029f: [6, 16],
	0x02a0: [6, 16],
	0x02b6: [6, 16],
	0x02b8: [6, 16],
	0x02c5: [6, 16],
	// classic 6×25 (Blade Pro 2017)
	0x0225: [6, 25],
	0x022f: [6, 25],
	// classic 6×17 / 6×19 (recent Blade generations)
	0x02c6: [6, 17],
	0x02c7: [6, 19],
	// extended TKL - DeathStalker V2 Pro TKL drops the cluster, not just the numpad
	0x0296: [6, 17],
	0x0298: [6, 17],
	// extended 75% - no nav cluster
	0x02a5: [6, 16],
	// extended full-size with extra rows (wrist-rest lightbar / macro row)
	0x0226: [9, 22],
	0x028d: [8, 23]
};

function defaultMatrix(style: MatrixStyle, layout: LayoutKind): [number, number] {
	if (style === 'extended') {
		if (layout === 'mini') return [5, 15];
		if (layout === 'tkl') return [6, 18];
		return [6, 22];
	}
	// classic: desktop Chroma family is 6×22, compact boards are one-row.
	return layout === 'mini' ? [1, 12] : [6, 22];
}

const RAW: RawSpec[] = [
	// classic matrix devices (style classic, report 1)
	{ pid: 0x0203, name: 'BlackWidow Chroma', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/279/279_blackwidow_chroma.png' },
	{
		pid: 0x0204,
		name: 'DeathStalker Chroma',
		report: 1,
		tid: 0x3f,
		style: 'classic',
		layout: 'mini',
		image: 'https://assets.razerzone.com/eeimages/support/products/665/665_deathstalker_chroma.png'
	},
	{ pid: 0x0205, name: 'Blade Stealth', report: 1, tid: 0xff, style: 'classic', layout: 'tkl', image: 'https://assets.razerzone.com/eeimages/support/products/667/667_blade_stealth_2016_6500u.png' },
	{
		pid: 0x0209,
		name: 'BlackWidow Chroma TE',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/571/571_blackwidow_tournament_edition_chroma.png'
	},
	{ pid: 0x020f, name: 'Blade QHD', report: 1, tid: 0xff, style: 'classic', layout: 'tkl', image: 'https://assets.razerzone.com/eeimages/support/products/736/736_blade_pro_2016.png' },
	{ pid: 0x0210, name: 'Blade Pro (Late 2016)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/736/736_blade_pro_2016.png' },
	{
		pid: 0x0211,
		name: 'BlackWidow Chroma (Overwatch)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/products/23326/overwatch-razer-gallery-5.png'
	},
	{
		pid: 0x0214,
		name: 'BlackWidow Ultimate 2016',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/616/616_blackwidow_ultimate_2016.png'
	},
	{
		pid: 0x0216,
		name: 'BlackWidow X Chroma',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/716/716_blackwidow_x_chroma.png'
	},
	{
		pid: 0x0217,
		name: 'BlackWidow X Ultimate',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/718/718_blackwidow_x_ultimate.png'
	},
	{
		pid: 0x021a,
		name: 'BlackWidow X Chroma TE',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/717/717_blackwidow_x_tournament_edition_chroma.png'
	},
	{
		pid: 0x0221,
		name: 'BlackWidow Chroma V2',
		report: 1,
		tid: 0x3f,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1179/1179_blackwidow_chroma_v2_alt.png'
	},
	{
		pid: 0x0220,
		name: 'Blade Stealth (Late 2016)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/667/667_blade_stealth_2016_6500u.png'
	},
	{ pid: 0x0224, name: 'Blade (Late 2016)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/736/736_blade_pro_2016.png' },
	{ pid: 0x0225, name: 'Blade Pro (2017)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1200/1200_blade_pro_2017.png' },
	{
		pid: 0x022d,
		name: 'Blade Stealth (Mid 2017)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/1213/1213_blade_stealth_2017_7500u.png'
	},
	{
		pid: 0x022f,
		name: 'Blade Pro FullHD (2017)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1200/1200_blade_pro_2017.png'
	},
	{
		pid: 0x0232,
		name: 'Blade Stealth (Late 2017)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/1213/1213_blade_stealth_2017_7500u.png'
	},
	{ pid: 0x0233, name: 'Blade 15 (2018)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1418/1418_blade_2018__base.png' },
	{
		pid: 0x0234,
		name: 'Blade Pro 17 (2019)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/razer-blade-pro-17/razer-blade-pro-17-2019-OGimage-1200x630.jpg'
	},
	{ pid: 0x0237, name: 'BlackWidow Essential', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1501/1501-blackwidow2019.png' },
	{
		pid: 0x0239,
		name: 'Blade Stealth (2019)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/1475/1475_bladestealth13(2019).png'
	},
	{ pid: 0x023a, name: 'Blade 15 Advanced (2019)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1482/blade15.png' },
	{
		pid: 0x023b,
		name: 'Blade 15 Base (2018)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1418/1418_blade_2018__base.png'
	},
	{
		pid: 0x0240,
		name: 'Blade 15 Mercury (2018)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1552/1552-blade-stealth-mercury-white.png'
	},
	{ pid: 0x0245, name: 'Blade 15 Mercury (Mid 2019)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets2.razerzone.com/images/blade-15/shop/blade15-mercury-1.jpg' },
	{
		pid: 0x0246,
		name: 'Blade 15 Base (Mid 2019)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1518/1518_blade15_mid2019-base.png'
	},
	{ pid: 0x024a, name: 'Blade Stealth (Late 2019)', report: 1, tid: 0xff, style: 'classic', layout: 'tkl', image: 'https://assets2.razerzone.com/images/blade-stealth-13/shop/stealth-l2p-1.jpg' },
	{
		pid: 0x024b,
		name: 'Blade Advanced (Late 2019)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/razer-blade-pro-17/razer-blade-pro-17-2019-OGimage-1200x630.jpg'
	},
	{
		pid: 0x024c,
		name: 'Blade Pro (Late 2019)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/razer-blade-pro-17/razer-blade-pro-17-2019-OGimage-1200x630.jpg'
	},
	{ pid: 0x024d, name: 'Blade 15 Studio (2019)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets2.razerzone.com/images/blade-15/shop/studio-ch41-1.jpg' },
	{ pid: 0x0252, name: 'Blade Stealth (Early 2020)', report: 1, tid: 0xff, style: 'classic', layout: 'tkl', image: 'https://assets2.razerzone.com/images/blade-stealth-13/shop/sl25p-fhd-4.jpg' },
	{
		pid: 0x0253,
		name: 'Blade 15 Advanced (2020)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1651/razer-blade-15-advanced-2020.png'
	},
	{
		pid: 0x0255,
		name: 'Blade 15 Base (Early 2020)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/blade-15/blade-15-base-model-spec-image-v2.png'
	},
	{
		pid: 0x0256,
		name: 'Blade Pro 17 (Early 2020)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1654/blade-pro-17-2020-2.png'
	},
	{ pid: 0x0259, name: 'Blade Stealth (Late 2020)', report: 1, tid: 0xff, style: 'classic', layout: 'tkl', image: 'https://assets2.razerzone.com/images/blade-stealth-13/shop/sl25p-fhd-4.jpg' },
	{
		pid: 0x0268,
		name: 'Blade 15 Base (Late 2020)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/blade-15/shop/blade15-base-model-spec-image-v2.png'
	},
	{ pid: 0x026a, name: 'Book 13 (2020)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1743/razerbook132020.png' },
	{
		pid: 0x026d,
		name: 'Blade 15 Advanced (Early 2021)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1761/blade-15-advanced-2021-rz09-0367x.png'
	},
	{ pid: 0x026e, name: 'Blade 17 Pro (Early 2021)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src/4025-16-EN-v1.png' },
	{
		pid: 0x026f,
		name: 'Blade 15 Base (Early 2021)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1756/blade-15-base-2021-rz09-0369x.png'
	},
	{ pid: 0x0270, name: 'Blade 14 (2021)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://assets2.razerzone.com/images/og-image/razer-blade-14-og-image-1200x630.jpg' },
	{
		pid: 0x0276,
		name: 'Blade 15 Advanced (Mid 2021)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1778/1778-razerblade15advanced2021rz09-0409x-2.png'
	},
	{ pid: 0x0279, name: 'Blade 17 Pro (Mid 2021)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src/5524/5524-1-en-v2.png' },
	{
		pid: 0x027a,
		name: 'Blade 15 Base (2022)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets.razerzone.com/eeimages/support/products/1756/blade-15-base-2021-rz09-0369x.png'
	},
	{
		pid: 0x028a,
		name: 'Blade 15 Advanced (Early 2022)',
		report: 1,
		tid: 0xff,
		style: 'classic',
		layout: 'full',
		image: 'https://assets2.razerzone.com/images/pnx.assets/a1ee4c5a780a401444be898fe93ade69/thumbnail-blade15-new-model.png'
	},
	{ pid: 0x028b, name: 'Blade 17 (2022)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src/5896/5896-1-en-v2.png' },
	{ pid: 0x028c, name: 'Blade 14 (2022)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src/5896/5896-1-en-v2.png' },
	{ pid: 0x029d, name: 'Blade 14 (2023)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/13031/13031-1-en-v2.png' },
	{ pid: 0x029e, name: 'Blade 15 (2023)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/9696/9696-1-en-v2.png' },
	{ pid: 0x029f, name: 'Blade 16 (2023)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src/9668/9668-1-en-v2.png' },
	{ pid: 0x02a0, name: 'Blade 18 (2023)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/9676/9676-1-en-v1.png' },
	{ pid: 0x02b6, name: 'Blade 14 (2024)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/13031/13031-1-en-v2.png' },
	{ pid: 0x02b8, name: 'Blade 18 (2024)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/9676/9676-1-en-v1.png' },
	{ pid: 0x02c5, name: 'Blade 14 (2025)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/15088/15088-1-en-v1.png' },
	{ pid: 0x02c6, name: 'Blade 16 (2025)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/14806/14806-en-v1.png' },
	{ pid: 0x02c7, name: 'Blade 18 (2025)', report: 1, tid: 0xff, style: 'classic', layout: 'full', image: 'https://dl.razerzone.com/src2/14968/14968-1-en-v1.png' },

	// extended, report 2
	{ pid: 0x010f, name: 'Anansi', report: 2, tid: 0x1f, style: 'extended', layout: 'full', custom: false, image: 'https://assets.razerzone.com/eeimages/support/products/54/54_anansi.png' },
	{
		pid: 0x0208,
		name: 'Tartarus Chroma',
		report: 2,
		tid: 0x1f,
		style: 'classic',
		layout: 'mini',
		custom: false,
		image: 'https://assets.razerzone.com/eeimages/support/products/598/598_tartarus_chroma.png'
	},
	{ pid: 0x0226, name: 'Huntsman Elite', report: 2, tid: 0x3f, style: 'extended', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1361/1361_huntsman_elite.png' },
	{
		pid: 0x0243,
		name: 'Huntsman Tournament Edition',
		report: 2,
		tid: 0x3f,
		style: 'extended',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/1537/1537_huntsman_te.png'
	},
	{ pid: 0x0257, name: 'Huntsman Mini', report: 2, tid: 0x1f, style: 'extended', layout: 'mini', image: 'https://assets.razerzone.com/eeimages/support/products/1689/1689-huntsmanmini.png' },
	{ pid: 0x025d, name: 'Ornata V2', report: 2, tid: 0x1f, style: 'extended', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1672/ornata-v2.png' },
	{ pid: 0x0293, name: 'BlackWidow V4 X', report: 2, tid: 0x1f, style: 'extended', layout: 'full', wheel: true, image: 'https://dl.razerzone.com/src2/13223/13223-1-en-v1.png' },
	{ pid: 0x0228, name: 'BlackWidow Elite', report: 2, tid: 0x1f, style: 'extended', layout: 'full', image: 'https://assets.razerzone.com/eeimages/support/products/1398/1398_blackwidowelite.png' },

	// extended, report 3
	{ pid: 0x026b, name: 'Huntsman V2 Tenkeyless', report: 3, tid: 0x1f, style: 'extended', layout: 'tkl', image: 'https://dl.razerzone.com/src/5638/5638-1-en-v1.png' },
	{ pid: 0x026c, name: 'Huntsman V2', report: 3, tid: 0x1f, style: 'extended', layout: 'full', image: 'https://dl.razerzone.com/src/5642/5642-1-en-v1.png' },
	{ pid: 0x024e, name: 'BlackWidow V3', report: 3, tid: 0x3f, style: 'extended', layout: 'full', image: 'https://dl.razerzone.com/src/3827-1-EN-v1.png' },
	{
		pid: 0x0a24,
		name: 'BlackWidow V3 Tenkeyless',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'tkl',
		image: 'https://assets.razerzone.com/eeimages/support/products/1709/1709-blackwidow-v3-tkl.png'
	},
	{
		pid: 0x025a,
		name: 'BlackWidow V3 Pro (Wired)',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'full',
		battery: true,
		batteryTid: 0x3f,
		image: 'https://dl.razerzone.com/src/3809-1-EN-v1.png'
	},
	{
		pid: 0x025c,
		name: 'BlackWidow V3 Pro (Wireless)',
		report: 3,
		tid: 0x9f,
		style: 'extended',
		layout: 'full',
		battery: true,
		batteryTid: 0x9f,
		image: 'https://dl.razerzone.com/src/3809-1-EN-v1.png'
	},
	{ pid: 0x0295, name: 'DeathStalker V2', report: 3, tid: 0x3f, style: 'extended', layout: 'full', image: 'https://dl.razerzone.com/src/6118/6118-1-en-v1.png' },
	{
		pid: 0x0292,
		name: 'DeathStalker V2 Pro (Wired)',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'full',
		battery: true,
		batteryTid: 0x1f,
		image: 'https://dl.razerzone.com/src/6118/6118-1-en-v1.png'
	},
	{
		pid: 0x0290,
		name: 'DeathStalker V2 Pro (Wireless)',
		report: 3,
		tid: 0x9f,
		style: 'extended',
		layout: 'full',
		battery: true,
		batteryTid: 0x9f,
		image: 'https://dl.razerzone.com/src/6118/6118-1-en-v1.png'
	},
	{
		pid: 0x0298,
		name: 'DeathStalker V2 Pro TKL (Wired)',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'tkl',
		battery: true,
		batteryTid: 0x1f,
		image: 'https://dl.razerzone.com/src/6117/6117-1-en-v1.png'
	},
	{
		pid: 0x0296,
		name: 'DeathStalker V2 Pro TKL (Wireless)',
		report: 3,
		tid: 0x9f,
		style: 'extended',
		layout: 'tkl',
		battery: true,
		batteryTid: 0x9f,
		image: 'https://dl.razerzone.com/src/6117/6117-1-en-v1.png'
	},
	{ pid: 0x02a6, name: 'Huntsman V3 Pro', report: 3, tid: 0x3f, style: 'extended', layout: 'full', image: 'https://dl.razerzone.com/src2/13671/13671-1-en-v2.png' },
	{
		pid: 0x02a7,
		name: 'Huntsman V3 Pro TKL',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'tkl',
		image: 'https://assets2.razerzone.com/images/pnx.assets/d01af9710e2077e2b88eb9dc6f68f62f/quick-actuation-adjustment-mode.webp'
	},
	{
		pid: 0x02b0,
		name: 'Huntsman V3 Pro Mini',
		report: 3,
		tid: 0x3f,
		style: 'extended',
		layout: 'mini',
		image: 'https://medias-p1.phoenix.razer.com/sys-master-phoenix-images-container/h8d/h31/9662065901598/230921-huntsman-v3-pro-mini-black-1500x1000-5.jpg'
	},
	{ pid: 0x028d, name: 'BlackWidow V4 Pro', report: 3, tid: 0x1f, style: 'extended', layout: 'full', wheel: true, image: 'https://dl.razerzone.com/src2/9703/9703-1-en-v1.png' },
	{ pid: 0x02a5, name: 'BlackWidow V4 75%', report: 3, tid: 0x1f, style: 'extended', layout: 'tkl', wheel: true, image: 'https://dl.razerzone.com/src2/13256/13256-1-en-v2.png' }
];

const byPid = new Map<number, RawSpec>();
for (const spec of RAW) byPid.set(spec.pid, spec);

export const KEYBOARD_DEVICES: KeyboardDevice[] = [...byPid.values()].map((s) => {
	const [matrixRows, matrixCols] = MATRIX_OVERRIDES[s.pid] ?? defaultMatrix(s.style, s.layout ?? 'full');
	return {
		pid: s.pid,
		name: s.name,
		reportIndex: s.report,
		transactionId: s.tid,
		style: s.style,
		layout: s.layout ?? 'full',
		waveLeft: s.waveLeft,
		battery: s.battery,
		batteryTid: s.batteryTid,
		image: s.image ?? `/keyboards/0x${s.pid.toString(16).padStart(4, '0')}.png`,
		matrixRows,
		matrixCols,
		custom: s.custom ?? true,
		wheel: s.wheel
	};
});

export function getKeyboard(pid: number): KeyboardDevice | undefined {
	return KEYBOARD_DEVICES.find((d) => d.pid === pid);
}

export function pidLabel(pid: number, productName?: string): string {
	const dev = getKeyboard(pid);
	if (dev) return dev.name;
	return productName?.trim() ? productName.trim() : `Razer Device (0x${pid.toString(16).padStart(4, '0')})`;
}
