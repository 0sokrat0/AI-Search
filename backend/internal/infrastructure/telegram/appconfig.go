package telegram

import (
	"math/rand"

	"github.com/gotd/td/telegram"
)

type appCredentials struct {
	AppID   int
	AppHash string
	Device  telegram.DeviceConfig
}

var knownApps = []appCredentials{
	{
		AppID:   4,
		AppHash: "014b35b6184100b085b0d0572f9b5103",
		Device: telegram.DeviceConfig{
			DeviceModel:    "Samsung Galaxy S24",
			SystemVersion:  "SDK 35",
			AppVersion:     "11.7.3 (5277)",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   5,
		AppHash: "1c5c96d5edd401b1ed40db3fb5633e2d",
		Device: telegram.DeviceConfig{
			DeviceModel:    "Xiaomi 14",
			SystemVersion:  "SDK 34",
			AppVersion:     "11.7.3 (5277)",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   6,
		AppHash: "eb06d4abfb49dc3eeb1aeb98ae0f581e",
		Device: telegram.DeviceConfig{
			DeviceModel:    "POCO X6 Pro",
			SystemVersion:  "SDK 34",
			AppVersion:     "11.7.3 (5277)",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   8,
		AppHash: "7245de8e747a0d6fbe11f7cc14fcc0bb",
		Device: telegram.DeviceConfig{
			DeviceModel:    "iPhone 16 Pro",
			SystemVersion:  "18.3.2",
			AppVersion:     "11.7.3",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   2834,
		AppHash: "68875f756c9b437a8b916ca3de215815",
		Device: telegram.DeviceConfig{
			DeviceModel:    "MacBook Pro",
			SystemVersion:  "15.3.1",
			AppVersion:     "11.7.3",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   2040,
		AppHash: "b18441a1ff607e10a989891a5462e627",
		Device: telegram.DeviceConfig{
			DeviceModel:    "PC 64bit",
			SystemVersion:  "Windows 11 24H2",
			AppVersion:     "5.4.1 x64",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   21950768,
		AppHash: "c5d0095e54a3e4f9cdf669b390b39090",
		Device: telegram.DeviceConfig{
			DeviceModel:    "OnePlus 12",
			SystemVersion:  "SDK 35",
			AppVersion:     "11.7.3 (5277)",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
	{
		AppID:   26793495,
		AppHash: "e76ed299ee59c840de5aae8c02868efa",
		Device: telegram.DeviceConfig{
			DeviceModel:    "Google Pixel 9 Pro",
			SystemVersion:  "SDK 35",
			AppVersion:     "11.7.3 (5277)",
			LangCode:       "ru",
			SystemLangCode: "ru-RU",
		},
	},
}

func pickRandomApp() appCredentials {
	return knownApps[rand.Intn(len(knownApps))]
}
