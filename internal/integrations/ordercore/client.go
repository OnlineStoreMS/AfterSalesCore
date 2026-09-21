package ordercore

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

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) FenFaRemarks(ctx context.Context, token string, orderNos []string) (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("订单中心未配置")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("未登录订单中心")
	}
	nos := make([]string, 0, len(orderNos))
	seen := map[string]struct{}{}
	for _, raw := range orderNos {
		n := strings.TrimSpace(raw)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		nos = append(nos, n)
	}
	out := map[string]string{}
	if len(nos) == 0 {
		return out, nil
	}
	const chunk = 400
	for i := 0; i < len(nos); i += chunk {
		end := i + chunk
		if end > len(nos) {
			end = len(nos)
		}
		part, err := c.fenFaChunk(ctx, token, nos[i:end])
		if err != nil {
			return nil, err
		}
		for k, v := range part {
			out[k] = v
		}
	}
	return out, nil
}

func (c *Client) fenFaChunk(ctx context.Context, token string, orderNos []string) (map[string]string, error) {
	payload, err := json.Marshal(map[string]any{"orderNos": orderNos})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/admin/orders/fenfa-remarks", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询订单中心分发备注: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var body apiBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("订单中心响应无效: %s", strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 300 || body.Code != 200 {
		msg := strings.TrimSpace(body.Message)
		if msg == "" {
			msg = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("订单中心: %s", msg)
	}
	out := map[string]string{}
	if len(body.Data) == 0 || string(body.Data) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(body.Data, &out); err != nil {
		return nil, fmt.Errorf("解析分发备注失败: %w", err)
	}
	return out, nil
}
