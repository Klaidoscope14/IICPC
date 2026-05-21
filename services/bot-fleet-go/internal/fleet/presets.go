package fleet

import "github.com/iicpc/bot-fleet-go/internal/bot"

// Preset holds named benchmark configurations.
type Preset struct {
	Name            string
	BotConcurrency  int
	DurationSeconds int32
	OrdersPerSecond int32
	Profile         bot.OrderProfile
}

var DefaultProfile = bot.OrderProfile{
	LimitWeight:  50,
	MarketWeight: 30,
	CancelWeight: 20,
	BuyWeight:    50,
}

var presets = map[string]Preset{
	"low_volatility": {
		Name:            "low_volatility",
		BotConcurrency:  50,
		DurationSeconds: 60,
		OrdersPerSecond: 20,
		Profile: bot.OrderProfile{
			LimitWeight:  80,
			MarketWeight: 10,
			CancelWeight: 10,
			BuyWeight:    50,
		},
	},
	"medium_traffic": {
		Name:            "medium_traffic",
		BotConcurrency:  150,
		DurationSeconds: 60,
		OrdersPerSecond: 100,
		Profile:         DefaultProfile,
	},
	"high_frequency_burst": {
		Name:            "high_frequency_burst",
		BotConcurrency:  300,
		DurationSeconds: 30,
		OrdersPerSecond: 500,
		Profile: bot.OrderProfile{
			LimitWeight:  40,
			MarketWeight: 20,
			CancelWeight: 40,
			BuyWeight:    50,
		},
	},
	"market_open_chaos": {
		Name:            "market_open_chaos",
		BotConcurrency:  500,
		DurationSeconds: 45,
		OrdersPerSecond: 800,
		Profile: bot.OrderProfile{
			LimitWeight:  30,
			MarketWeight: 50,
			CancelWeight: 20,
			BuyWeight:    50,
		},
	},
	"flash_crash": {
		Name:            "flash_crash",
		BotConcurrency:  200,
		DurationSeconds: 30,
		OrdersPerSecond: 600,
		Profile: bot.OrderProfile{
			LimitWeight:  10,
			MarketWeight: 80,
			CancelWeight: 10,
			BuyWeight:    10, // 90% SELL orders
		},
	},
	"stress_overload": {
		Name:            "stress_overload",
		BotConcurrency:  1000,
		DurationSeconds: 60,
		OrdersPerSecond: 2000,
		Profile:         DefaultProfile,
	},
}

// GetPreset returns the named preset. If not found, it returns medium_traffic default.
func GetPreset(name string) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets["medium_traffic"]
}

// PresetNames returns all available preset names.
func PresetNames() []string {
	names := make([]string, 0, len(presets))
	for k := range presets {
		names = append(names, k)
	}
	return names
}
