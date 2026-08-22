package dto

type PluginDebugEvent struct {
	Ms    int64       `json:"ms"`
	At    string      `json:"at"`
	Level string      `json:"level"`
	Step  string      `json:"step"`
	Data  interface{} `json:"data,omitempty"`
}

type PluginDebugLogInput struct {
	RunID      string                 `json:"runId"`
	Kind       string                 `json:"kind"`
	OK         bool                   `json:"ok"`
	Error      string                 `json:"error"`
	DurationMs int64                  `json:"durationMs"`
	Version    string                 `json:"version"`
	Meta       map[string]interface{} `json:"meta"`
	Events     []PluginDebugEvent     `json:"events"`
}

type PluginDebugLogResult struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}
