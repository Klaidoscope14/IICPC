package fleet

// Preset holds named benchmark configurations.
type Preset struct {
	Name            string
	BotConcurrency  int
	DurationSeconds int32
	OrdersPerSecond int32
}

var presets = map[string]Preset{
	"low_load": {
		Name:            "low_load",
		BotConcurrency:  50,
		DurationSeconds: 30,
		OrdersPerSecond: 20,
	},
	"burst": {
		Name:            "burst",
		BotConcurrency:  200,
		DurationSeconds: 30,
		OrdersPerSecond: 200,
	},
	"chaos": {
		Name:            "chaos",
		BotConcurrency:  500,
		DurationSeconds: 30,
		OrdersPerSecond: 500,
	},
}

// GetPreset returns the named preset, or the low_load default.
func GetPreset(name string) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets["low_load"]
}

// PresetNames returns all available preset names.
func PresetNames() []string {
	names := make([]string, 0, len(presets))
	for k := range presets {
		names = append(names, k)
	}
	return names
}
