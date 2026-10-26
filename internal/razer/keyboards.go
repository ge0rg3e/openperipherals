package razer

// Model describes a supported keyboard. The table is OpenRazer's, as
// collected by the OpenPeripherals web app.
type Model struct {
	PID           uint16
	Name          string
	TransactionID byte
	Extended      bool   // extended matrix (class 0x0f) rather than classic (0x03)
	Interface     int    // USB interface that carries the control report
	Layout        string // "full", "tkl" or "mini"
	Rows, Cols    int    // LED matrix for custom frames
	Custom        bool   // accepts host-driven custom frames
	Wheel         bool   // hardware colour wheel (BlackWidow V4 family)
	Battery       bool
	BatteryTID    byte
	Wireless      bool // replies take longer over the air
}

type kbSpec struct {
	pid       uint16
	name      string
	iface     int
	tid       byte
	ext       bool
	layout    string
	noCustom  bool
	wheel     bool
	batteryID byte // non-zero for boards with a battery
}

// Matrix sizes that differ from the layout default (OpenRazer MATRIX_DIMS).
var matrixOverrides = map[uint16][2]int{
	// classic 6x16 laptop matrix (Blade family)
	0x020f: {6, 16}, 0x0220: {6, 16}, 0x0224: {6, 16}, 0x022d: {6, 16}, 0x0232: {6, 16},
	0x0233: {6, 16}, 0x0234: {6, 16}, 0x0239: {6, 16}, 0x023a: {6, 16}, 0x023b: {6, 16},
	0x0240: {6, 16}, 0x0245: {6, 16}, 0x0246: {6, 16}, 0x024a: {6, 16}, 0x024b: {6, 16},
	0x024c: {6, 16}, 0x024d: {6, 16}, 0x0252: {6, 16}, 0x0253: {6, 16}, 0x0255: {6, 16},
	0x0256: {6, 16}, 0x0259: {6, 16}, 0x0268: {6, 16}, 0x026a: {6, 16}, 0x026d: {6, 16},
	0x026e: {6, 16}, 0x026f: {6, 16}, 0x0270: {6, 16}, 0x0276: {6, 16}, 0x0279: {6, 16},
	0x027a: {6, 16}, 0x028a: {6, 16}, 0x028b: {6, 16}, 0x028c: {6, 16}, 0x029d: {6, 16},
	0x029e: {6, 16}, 0x029f: {6, 16}, 0x02a0: {6, 16}, 0x02b6: {6, 16}, 0x02b8: {6, 16},
	0x02c5: {6, 16},
	// classic 6x25 (Blade Pro 2017)
	0x0225: {6, 25}, 0x022f: {6, 25},
	// classic 6x17 / 6x19 (recent Blades)
	0x02c6: {6, 17}, 0x02c7: {6, 19},
	// extended TKL without the nav cluster
	0x0296: {6, 17}, 0x0298: {6, 17},
	// extended 75%
	0x02a5: {6, 16},
	// extra rows: wrist-rest lightbar / macro row
	0x0226: {9, 22}, 0x028d: {8, 23},
}

func blade(pid uint16, name, layout string) kbSpec {
	return kbSpec{pid: pid, name: name, iface: 1, tid: 0xff, layout: layout}
}

var kbSpecs = []kbSpec{
	// classic matrix
	{pid: 0x0203, name: "BlackWidow Chroma", iface: 1, tid: 0xff, layout: "full"},
	{pid: 0x0204, name: "DeathStalker Chroma", iface: 1, tid: 0x3f, layout: "mini"},
	blade(0x0205, "Blade Stealth", "tkl"),
	{pid: 0x0209, name: "BlackWidow Chroma TE", iface: 1, tid: 0xff, layout: "tkl"},
	blade(0x020f, "Blade QHD", "tkl"),
	blade(0x0210, "Blade Pro (Late 2016)", "full"),
	{pid: 0x0211, name: "BlackWidow Chroma (Overwatch)", iface: 1, tid: 0xff, layout: "full"},
	{pid: 0x0214, name: "BlackWidow Ultimate 2016", iface: 1, tid: 0xff, layout: "full"},
	{pid: 0x0216, name: "BlackWidow X Chroma", iface: 1, tid: 0xff, layout: "full"},
	{pid: 0x0217, name: "BlackWidow X Ultimate", iface: 1, tid: 0xff, layout: "full"},
	{pid: 0x021a, name: "BlackWidow X Chroma TE", iface: 1, tid: 0xff, layout: "tkl"},
	{pid: 0x0221, name: "BlackWidow Chroma V2", iface: 1, tid: 0x3f, layout: "full"},
	blade(0x0220, "Blade Stealth (Late 2016)", "tkl"),
	blade(0x0224, "Blade (Late 2016)", "full"),
	blade(0x0225, "Blade Pro (2017)", "full"),
	blade(0x022d, "Blade Stealth (Mid 2017)", "tkl"),
	blade(0x022f, "Blade Pro FullHD (2017)", "full"),
	blade(0x0232, "Blade Stealth (Late 2017)", "tkl"),
	blade(0x0233, "Blade 15 (2018)", "full"),
	blade(0x0234, "Blade Pro 17 (2019)", "full"),
	{pid: 0x0237, name: "BlackWidow Essential", iface: 1, tid: 0xff, layout: "full"},
	blade(0x0239, "Blade Stealth (2019)", "tkl"),
	blade(0x023a, "Blade 15 Advanced (2019)", "full"),
	blade(0x023b, "Blade 15 Base (2018)", "full"),
	blade(0x0240, "Blade 15 Mercury (2018)", "full"),
	blade(0x0245, "Blade 15 Mercury (Mid 2019)", "full"),
	blade(0x0246, "Blade 15 Base (Mid 2019)", "full"),
	blade(0x024a, "Blade Stealth (Late 2019)", "tkl"),
	blade(0x024b, "Blade Advanced (Late 2019)", "full"),
	blade(0x024c, "Blade Pro (Late 2019)", "full"),
	blade(0x024d, "Blade 15 Studio (2019)", "full"),
	blade(0x0252, "Blade Stealth (Early 2020)", "tkl"),
	blade(0x0253, "Blade 15 Advanced (2020)", "full"),
	blade(0x0255, "Blade 15 Base (Early 2020)", "full"),
	blade(0x0256, "Blade Pro 17 (Early 2020)", "full"),
	blade(0x0259, "Blade Stealth (Late 2020)", "tkl"),
	blade(0x0268, "Blade 15 Base (Late 2020)", "full"),
	blade(0x026a, "Book 13 (2020)", "full"),
	blade(0x026d, "Blade 15 Advanced (Early 2021)", "full"),
	blade(0x026e, "Blade 17 Pro (Early 2021)", "full"),
	blade(0x026f, "Blade 15 Base (Early 2021)", "full"),
	blade(0x0270, "Blade 14 (2021)", "full"),
	blade(0x0276, "Blade 15 Advanced (Mid 2021)", "full"),
	blade(0x0279, "Blade 17 Pro (Mid 2021)", "full"),
	blade(0x027a, "Blade 15 Base (2022)", "full"),
	blade(0x028a, "Blade 15 Advanced (Early 2022)", "full"),
	blade(0x028b, "Blade 17 (2022)", "full"),
	blade(0x028c, "Blade 14 (2022)", "full"),
	blade(0x029d, "Blade 14 (2023)", "full"),
	blade(0x029e, "Blade 15 (2023)", "full"),
	blade(0x029f, "Blade 16 (2023)", "full"),
	blade(0x02a0, "Blade 18 (2023)", "full"),
	blade(0x02b6, "Blade 14 (2024)", "full"),
	blade(0x02b8, "Blade 18 (2024)", "full"),
	blade(0x02c5, "Blade 14 (2025)", "full"),
	blade(0x02c6, "Blade 16 (2025)", "full"),
	blade(0x02c7, "Blade 18 (2025)", "full"),

	// interface 2
	{pid: 0x010f, name: "Anansi", iface: 2, tid: 0x1f, ext: true, layout: "full", noCustom: true},
	{pid: 0x0208, name: "Tartarus Chroma", iface: 2, tid: 0x1f, layout: "mini", noCustom: true},
	{pid: 0x0226, name: "Huntsman Elite", iface: 2, tid: 0x3f, ext: true, layout: "full"},
	{pid: 0x0243, name: "Huntsman Tournament Edition", iface: 2, tid: 0x3f, ext: true, layout: "tkl"},
	{pid: 0x0257, name: "Huntsman Mini", iface: 2, tid: 0x1f, ext: true, layout: "mini"},
	{pid: 0x025d, name: "Ornata V2", iface: 2, tid: 0x1f, ext: true, layout: "full"},
	{pid: 0x0293, name: "BlackWidow V4 X", iface: 2, tid: 0x1f, ext: true, layout: "full", wheel: true},
	{pid: 0x0228, name: "BlackWidow Elite", iface: 2, tid: 0x1f, ext: true, layout: "full"},

	// interface 3
	{pid: 0x026b, name: "Huntsman V2 Tenkeyless", iface: 3, tid: 0x1f, ext: true, layout: "tkl"},
	{pid: 0x026c, name: "Huntsman V2", iface: 3, tid: 0x1f, ext: true, layout: "full"},
	{pid: 0x024e, name: "BlackWidow V3", iface: 3, tid: 0x3f, ext: true, layout: "full"},
	{pid: 0x0a24, name: "BlackWidow V3 Tenkeyless", iface: 3, tid: 0x3f, ext: true, layout: "tkl"},
	{pid: 0x025a, name: "BlackWidow V3 Pro (Wired)", iface: 3, tid: 0x3f, ext: true, layout: "full", batteryID: 0x3f},
	{pid: 0x025c, name: "BlackWidow V3 Pro (Wireless)", iface: 3, tid: 0x9f, ext: true, layout: "full", batteryID: 0x9f},
	{pid: 0x0295, name: "DeathStalker V2", iface: 3, tid: 0x3f, ext: true, layout: "full"},
	{pid: 0x0292, name: "DeathStalker V2 Pro (Wired)", iface: 3, tid: 0x3f, ext: true, layout: "full", batteryID: 0x1f},
	{pid: 0x0290, name: "DeathStalker V2 Pro (Wireless)", iface: 3, tid: 0x9f, ext: true, layout: "full", batteryID: 0x9f},
	{pid: 0x0298, name: "DeathStalker V2 Pro TKL (Wired)", iface: 3, tid: 0x3f, ext: true, layout: "tkl", batteryID: 0x1f},
	{pid: 0x0296, name: "DeathStalker V2 Pro TKL (Wireless)", iface: 3, tid: 0x9f, ext: true, layout: "tkl", batteryID: 0x9f},
	{pid: 0x02a6, name: "Huntsman V3 Pro", iface: 3, tid: 0x3f, ext: true, layout: "full"},
	{pid: 0x02a7, name: "Huntsman V3 Pro TKL", iface: 3, tid: 0x3f, ext: true, layout: "tkl"},
	{pid: 0x02b0, name: "Huntsman V3 Pro Mini", iface: 3, tid: 0x3f, ext: true, layout: "mini"},
	{pid: 0x028d, name: "BlackWidow V4 Pro", iface: 3, tid: 0x1f, ext: true, layout: "full", wheel: true},
	{pid: 0x02a5, name: "BlackWidow V4 75%", iface: 3, tid: 0x1f, ext: true, layout: "tkl", wheel: true},
}

// Models lists the supported keyboards.
var Models = func() []Model {
	out := make([]Model, 0, len(kbSpecs))
	for _, s := range kbSpecs {
		rows, cols := 6, 22
		switch {
		case s.ext && s.layout == "mini":
			rows, cols = 5, 15
		case s.ext && s.layout == "tkl":
			rows, cols = 6, 18
		case !s.ext && s.layout == "mini":
			rows, cols = 1, 12
		}
		if d, ok := matrixOverrides[s.pid]; ok {
			rows, cols = d[0], d[1]
		}
		out = append(out, Model{
			PID: s.pid, Name: "Razer " + s.name, TransactionID: s.tid, Extended: s.ext, Interface: s.iface,
			Layout: s.layout, Rows: rows, Cols: cols, Custom: !s.noCustom, Wheel: s.wheel,
			Battery: s.batteryID != 0, BatteryTID: s.batteryID, Wireless: s.tid == 0x9f,
		})
	}
	return out
}()

var modelIndex = func() map[uint16]int {
	m := make(map[uint16]int, len(Models))
	for i, k := range Models {
		m[k.PID] = i
	}
	return m
}()

func keyboardModel(pid uint16) (Model, bool) {
	i, ok := modelIndex[pid]
	if !ok {
		return Model{}, false
	}
	return Models[i], true
}
