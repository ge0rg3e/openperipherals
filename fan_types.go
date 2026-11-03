package main

// FanConfig is how one fan is driven.
type FanConfig struct {
	Mode   string   `json:"mode"` // "auto", "fixed" or "curve"
	Fixed  int      `json:"fixed"`
	Sensor string   `json:"sensor"`
	Points [][2]int `json:"points"` // (°C, %) pairs, sorted by temperature
}

// Fan is one controllable fan as shown in the UI.
type Fan struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Source  string    `json:"source"`
	RPM     int       `json:"rpm"`  // -1 when the fan has no tachometer
	Duty    int       `json:"duty"` // -1 while firmware controls it
	MinDuty int       `json:"minDuty"`
	Config  FanConfig `json:"config"`
}

// TempSensor is a temperature a curve can follow.
type TempSensor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Temp int    `json:"temp"` // -1 when unavailable
}

// FanPreset is a named set of fan configs that can be applied at once.
type FanPreset struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Builtin bool             `json:"builtin"`
	Configs []FanPresetEntry `json:"configs"`
}

// FanPresetEntry is the config a preset applies to one fan.
type FanPresetEntry struct {
	FanID  string    `json:"fanId"`
	Config FanConfig `json:"config"`
}

// FanStatus is the fan page snapshot.
type FanStatus struct {
	Available bool         `json:"available"`
	Notice    string       `json:"notice"` // why motherboard fans are missing, if they are
	Elevated  bool         `json:"elevated"`
	Fans      []Fan        `json:"fans"`
	Sensors   []TempSensor `json:"sensors"`
	Presets   []FanPreset  `json:"presets"`
}
