package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AgentsCenterClient creates aftersale scrape jobs on AgentsCenter.
type AgentsCenterClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewAgentsCenterClient(baseURL, token string) *AgentsCenterClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(token) == "" {
		return nil
	}
	return &AgentsCenterClient{
		BaseURL: baseURL,
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

type agentsCreateJobBody struct {
	TenantID         uint64 `json:"tenantId"`
	JobType          string `json:"jobType"`
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	ParamsJSON       string `json:"paramsJson"`
	Source           string `json:"source"`
	Priority         int    `json:"priority"`
}

type AgentsOnlineShop struct {
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	BrowserChannel   string `json:"browserChannel"`
	AgentID          uint64 `json:"agentId"`
	AgentName        string `json:"agentName"`
	AgentOnline      bool   `json:"agentOnline"`
}

func (c *AgentsCenterClient) CreateAftersaleJob(tenantID uint64, platform, platformShopID, platformShopName, paramsJSON string) error {
	return c.CreateJob(tenantID, "doudian.aftersale", platform, platformShopID, platformShopName, paramsJSON, "aftersales")
}

func (c *AgentsCenterClient) CreateJob(tenantID uint64, jobType, platform, platformShopID, platformShopName, paramsJSON, source string) error {
	if c == nil {
		return fmt.Errorf("AgentsCenter 未配置")
	}
	if source == "" {
		source = "aftersales"
	}
	body := agentsCreateJobBody{
		TenantID:         tenantID,
		JobType:          jobType,
		Platform:         platform,
		PlatformShopID:   platformShopID,
		PlatformShopName: platformShopName,
		ParamsJSON:       paramsJSON,
		Source:           source,
		Priority:         100,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/internal/jobs", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.Token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("AgentsCenter HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *AgentsCenterClient) ListOnlineShops(tenantID uint64, platform string) ([]AgentsOnlineShop, error) {
	if c == nil {
		return nil, fmt.Errorf("AgentsCenter 未配置")
	}
	u := fmt.Sprintf("%s/api/v1/internal/shops?tenantId=%d&onlineOnly=1&page=1&pageSize=200", c.BaseURL, tenantID)
	if p := strings.TrimSpace(platform); p != "" {
		u += "&platform=" + url.QueryEscape(p)
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Token", c.Token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("AgentsCenter HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var envelope struct {
		Data struct {
			List []AgentsOnlineShop `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data.List, nil
}
