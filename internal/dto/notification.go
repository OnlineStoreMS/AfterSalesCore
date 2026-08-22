package dto

type NotificationConfig struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	Secret              string   `json:"secret,omitempty"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes,omitempty"`
	Scenarios           []string `json:"scenarios,omitempty"`
	ShopIDs             []string `json:"shopIds,omitempty"`
	AppID               string   `json:"appId,omitempty"`
	AppSecret           string   `json:"appSecret,omitempty"`
}

type NotificationState struct {
	LastRunAt        string            `json:"lastRunAt,omitempty"`
	LastRunOK        bool              `json:"lastRunOk"`
	LastError        string            `json:"lastError,omitempty"`
	LastSentCount    int               `json:"lastSentCount,omitempty"`
	LastBarcodeError string            `json:"lastBarcodeError,omitempty"`
	Notified         map[string]string `json:"notified,omitempty"`
}

type NotificationData struct {
	Config NotificationConfig `json:"config"`
	State  NotificationState  `json:"state"`
}

type NotificationConfigView struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	SecretSet           bool     `json:"secretSet"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes"`
	Scenarios           []string `json:"scenarios"`
	ShopIDs             []string `json:"shopIds"`
	AppID               string   `json:"appId,omitempty"`
	AppSecretSet        bool     `json:"appSecretSet"`
}

type ScenarioOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group,omitempty"`
}

type NotificationShopOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotificationView struct {
	Config    NotificationConfigView   `json:"config"`
	State     NotificationState        `json:"state"`
	Scenarios []ScenarioOption         `json:"scenarios"`
	Shops     []NotificationShopOption `json:"shops"`
}

type NotificationRunResult struct {
	Sent             int    `json:"sent"`
	Skipped          int    `json:"skipped"`
	BarcodeWarnings  int    `json:"barcodeWarnings,omitempty"`
	LastBarcodeError string `json:"lastBarcodeError,omitempty"`
	Error            string `json:"error,omitempty"`
}
