package razer

// MouseModel describes a supported mouse. Razer's protocol isn't
// self-describing, so capabilities are kept per product ID: a wrong
// transaction ID means silence, a wrong ceiling is caught by the read-back
// every setter does, and a missing flag only hides a control.
type MouseModel struct {
	PID             uint16
	Name            string
	Wireless        bool
	PollingRates    []int
	MaxDPI          int
	TransactionID   byte
	Battery         bool
	DPIStore        byte // storage selector for DPI commands
	HighRatePolling bool // extended (divisor of 8000) polling command
	LiftOff         bool
}

var (
	rates1K = []int{125, 500, 1000}
	rates8K = []int{125, 500, 1000, 2000, 4000, 8000}
)

// Sensor ceilings by generation.
const (
	dpiChroma   = 16000
	dpiFocus    = 20000
	dpiFocusPro = 30000
	dpi35K      = 35000
)

// Transaction IDs audited per product; they follow no family or connection
// pattern. Everything else uses 0xff.
var (
	tid3F = []uint16{
		0x0050, 0x0059, 0x005a, 0x005c, 0x0060, 0x0064, 0x0065, 0x006f, 0x0070,
		0x0072, 0x0073, 0x007c, 0x007d, 0x0084, 0x008c,
		0x007a, 0x007b, // Viper Ultimate: observed answering 0x3f
		0x006e, 0x0071, 0x0098, // DeathAdder Essential
	}
	tid1F = []uint16{
		0x0062, 0x006c, 0x0077, 0x0080, 0x0085, 0x0086, 0x0088, 0x008d, 0x008f,
		0x0090, 0x0094, 0x0096, 0x0099, 0x009a, 0x009c, 0x009e, 0x009f, 0x00a1,
		0x00a4, 0x00a5, 0x00a6, 0x00a7, 0x00a8, 0x00aa, 0x00ab, 0x00af, 0x00b0, 0x00b2, 0x00b3,
		0x00b4, 0x00b6, 0x00b7, 0x00b8, 0x00b9, 0x00be, 0x00bf, 0x00c0, 0x00c1,
		0x00c2, 0x00c3, 0x00c4, 0x00c5, 0x00c7, 0x00c8, 0x00cb, 0x00cc, 0x00cd,
		0x00d0, 0x00d1, 0x00d3, 0x00d4, 0x00d6, 0x00d7, 0x00de, 0x00df,
	}
)

// Product families.
func standard(dpi int) MouseModel {
	return MouseModel{PollingRates: rates1K, MaxDPI: dpi, DPIStore: varStore}
}

func standardWireless(dpi int) MouseModel {
	m := standard(dpi)
	m.Wireless, m.Battery = true, true
	return m
}

// modernWired is the HyperSpeed generation on its cable.
func modernWired(dpi int) MouseModel {
	return MouseModel{PollingRates: rates1K, MaxDPI: dpi, Battery: true, DPIStore: varStore}
}

func modernReceiver(dpi int) MouseModel {
	m := modernWired(dpi)
	m.Wireless, m.HighRatePolling = true, true
	return m
}

// legacyReceiver predates the extended polling command.
func legacyReceiver(dpi int) MouseModel {
	m := modernReceiver(dpi)
	m.HighRatePolling = false
	return m
}

func with(m MouseModel, f func(*MouseModel)) MouseModel { f(&m); return m }

var (
	rate8K   = func(m *MouseModel) { m.PollingRates, m.HighRatePolling = rates8K, true }
	liftOff  = func(m *MouseModel) { m.LiftOff = true }
	legacyHz = func(m *MouseModel) { m.HighRatePolling = false }
)

var mouseList = []struct {
	pid  uint16
	name string
	m    MouseModel
}{
	// Viper V2/V3 Pro
	{0x00a5, "Viper V2 Pro", with(modernWired(dpiFocusPro), liftOff)},
	{0x00a6, "Viper V2 Pro", with(modernReceiver(dpiFocusPro), liftOff)},
	{0x00c0, "Viper V3 Pro", with(modernWired(dpi35K), liftOff)},
	{0x00c1, "Viper V3 Pro", with(modernReceiver(dpi35K), func(m *MouseModel) { rate8K(m); liftOff(m) })},

	// DeathAdder Essential
	{0x006e, "DeathAdder Essential", standard(6400)},
	{0x0071, "DeathAdder Essential White Edition", standard(6400)},
	{0x0098, "DeathAdder Essential (2021)", standard(6400)},

	// wired
	{0x0015, "Naga", standard(8200)},
	{0x0020, "Abyssus 1800", standard(1800)},
	{0x002e, "Naga 2012", standard(8200)},
	{0x0034, "Taipan", standard(8200)},
	{0x0037, "DeathAdder 2013", standard(6400)},
	{0x0040, "Naga 2014", standard(8200)},
	{0x0043, "DeathAdder Chroma", standard(dpiChroma)},
	{0x0046, "Mamba Tournament Edition", standard(dpiChroma)},
	{0x0053, "Naga Chroma", standard(dpiChroma)},
	{0x005c, "DeathAdder Elite", standard(dpiChroma)},
	{0x0060, "Lancehead Tournament Edition", standard(dpiChroma)},
	{0x0064, "Basilisk", standard(dpiChroma)},
	{0x0065, "Basilisk Essential", standard(6400)},
	{0x0067, "Naga Trinity", standard(dpiChroma)},
	{0x006c, "Mamba Elite", standard(dpiChroma)},
	{0x0084, "DeathAdder V2", with(standard(dpiFocus), func(m *MouseModel) { m.DPIStore = 0x00 })},
	{0x0085, "Basilisk V2", standard(dpiFocus)},
	{0x008c, "DeathAdder V2 Mini", standard(8500)},
	{0x0091, "Viper 8KHz", with(standard(dpiFocus), rate8K)},
	{0x00a1, "DeathAdder V2 Lite", standard(8500)},
	{0x00b2, "DeathAdder V3", with(standard(dpiFocusPro), rate8K)},
	{0x0096, "Naga X", standard(18000)},
	{0x0099, "Basilisk V3", standard(26000)},
	{0x00cb, "Basilisk V3 35K", standard(dpi35K)},

	// wireless
	{0x0062, "Atheris", standardWireless(7200)},
	{0x0094, "Orochi V2", standardWireless(18000)},
	{0x007a, "Viper Ultimate (Wired)", modernWired(dpiFocus)},
	{0x007b, "Viper Ultimate", modernReceiver(dpiFocus)},
	{0x007c, "DeathAdder V2 Pro (Wired)", modernWired(dpiFocus)},
	{0x007d, "DeathAdder V2 Pro", modernReceiver(dpiFocus)},
	{0x009e, "Viper Mini Signature Edition (Wired)", modernWired(dpiFocusPro)},
	{0x009f, "Viper Mini Signature Edition", with(modernReceiver(dpiFocusPro), func(m *MouseModel) { m.PollingRates = rates8K })},
	// The stock HyperSpeed receiver answers only the legacy polling command.
	{0x00b8, "Viper V3 HyperSpeed", with(modernReceiver(dpiFocusPro), func(m *MouseModel) { legacyHz(m); liftOff(m) })},
	{0x006f, "Lancehead Wireless", legacyReceiver(dpiFocus)},
	{0x0070, "Lancehead Wireless (Wired)", modernWired(dpiFocus)},
	{0x0072, "Mamba Wireless", legacyReceiver(dpiFocus)},
	{0x0077, "Pro Click", legacyReceiver(dpiFocus)},
	{0x0080, "Pro Click (Wired)", modernWired(dpiFocus)},
	{0x0083, "Basilisk X HyperSpeed", legacyReceiver(dpiFocus)},
	{0x0086, "Basilisk Ultimate (Wired)", modernWired(dpiFocus)},
	{0x0088, "Basilisk Ultimate", legacyReceiver(dpiFocus)},
	{0x008f, "Naga Pro (Wired)", modernWired(dpiFocus)},
	{0x0090, "Naga Pro", legacyReceiver(dpiFocus)},
	{0x009a, "Pro Click Mini", legacyReceiver(12000)},
	{0x009c, "DeathAdder V2 X HyperSpeed", legacyReceiver(14000)},
	{0x00a7, "Naga V2 Pro (Wired)", modernWired(dpiFocusPro)},
	{0x00a8, "Naga V2 Pro", modernReceiver(dpiFocusPro)},
	{0x00aa, "Basilisk V3 Pro (Wired)", modernWired(dpiFocusPro)},
	{0x00ab, "Basilisk V3 Pro", modernReceiver(dpiFocusPro)},
	{0x00af, "Cobra Pro (Wired)", modernWired(dpiFocusPro)},
	{0x00b0, "Cobra Pro", modernReceiver(dpiFocusPro)},
	{0x00b4, "Naga V2 HyperSpeed", legacyReceiver(dpiFocusPro)},
	{0x00b6, "DeathAdder V3 Pro (Wired)", modernWired(dpiFocusPro)},
	// The extended command reads back, but the stock receiver stays at 1000 Hz.
	{0x00b7, "DeathAdder V3 Pro", with(modernReceiver(dpiFocusPro), legacyHz)},
	{0x00b9, "Basilisk V3 X HyperSpeed", legacyReceiver(18000)},
	{0x00c2, "DeathAdder V3 Pro (Wired)", modernWired(dpiFocusPro)},
	{0x00c3, "DeathAdder V3 Pro", modernReceiver(dpiFocusPro)},
	{0x00c4, "DeathAdder V3 HyperSpeed (Wired)", modernWired(dpiFocusPro)},
	{0x00c5, "DeathAdder V3 HyperSpeed", legacyReceiver(dpiFocusPro)},
	{0x00cc, "Basilisk V3 Pro 35K (Wired)", modernWired(dpi35K)},
	{0x00cd, "Basilisk V3 Pro 35K", with(modernReceiver(dpi35K), func(m *MouseModel) { m.PollingRates = rates8K })},
}

var mouseModels = func() map[uint16]MouseModel {
	out := make(map[uint16]MouseModel, len(mouseList))
	for _, e := range mouseList {
		m := e.m
		m.PID, m.Name = e.pid, "Razer "+e.name
		m.TransactionID = 0xff
		for _, ids := range []struct {
			pids []uint16
			tid  byte
		}{{tid3F, 0x3f}, {tid1F, 0x1f}} {
			for _, p := range ids.pids {
				if p == e.pid {
					m.TransactionID = ids.tid
				}
			}
		}
		out[e.pid] = m
	}
	return out
}()
