package storesyncagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TradeGoods struct {
	Title   string `json:"title,omitempty"`
	SkuName string `json:"skuName,omitempty"`
	PicURL  string `json:"picUrl,omitempty"`
	Num     int    `json:"num,omitempty"`
}

type OrderLookup struct {
	Found      bool         `json:"found"`
	OrderNo    string       `json:"orderNo"`
	Platform   string       `json:"platform,omitempty"`
	ShopName   string       `json:"shopName,omitempty"`
	Goods      []TradeGoods `json:"goods,omitempty"`
	GoodsTitle string       `json:"goodsTitle,omitempty"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) LookupByTrackingNos(ctx context.Context, bearerToken, recordType string, trackingNos []string) (map[string]OrderLookup, error) {
	if !c.Enabled() || len(trackingNos) == 0 {
		return map[string]OrderLookup{}, nil
	}
	payload, err := json.Marshal(map[string]any{
		"trackingNos": trackingNos,
		"recordType":  recordType,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/orders/lookup-tracking", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storesyncagent request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("storesyncagent http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("storesyncagent decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "storesyncagent error"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	var data struct {
		Items map[string]*OrderLookup `json:"items"`
	}
	if err := json.Unmarshal(wrapped.Data, &data); err != nil {
		return nil, fmt.Errorf("storesyncagent data decode: %w", err)
	}

	out := make(map[string]OrderLookup, len(data.Items))
	for key, item := range data.Items {
		if item == nil {
			continue
		}
		out[strings.ToUpper(strings.TrimSpace(key))] = *item
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
