package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Password   string
	SessionID  string
	HTTPClient *http.Client
}

func NewClient(baseURL, password string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Authenticate() error {
	body := fmt.Sprintf(`{"password":"%s"}`, c.Password)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/auth", "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed with status %d", resp.StatusCode)
	}

	var result struct {
		Session struct {
			SID string `json:"sid"`
		} `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}
	c.SessionID = result.Session.SID
	return nil
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.BaseURL+"/api"+endpoint, body)
	if err != nil {
		return nil, err
	}
	if c.SessionID != "" {
		req.Header.Set("X-FTL-SID", c.SessionID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

type Summary struct {
	Queries       int     `json:"dns_queries_today"`
	Blocked       int     `json:"ads_blocked_today"`
	BlockedPct    float64 `json:"ads_percentage_today"`
	DomainsOnList int     `json:"domains_being_blocked"`
	Status        string  `json:"status"`
	ClientsEver   int     `json:"clients_ever_seen"`
	UniqueDomains int     `json:"unique_domains"`
	QueriesForwarded int  `json:"queries_forwarded"`
	QueriesCached int     `json:"queries_cached"`
}

type TopItem struct {
	Domain string
	Count  int
}

type QueryLogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Domain    string `json:"domain"`
	Client    string `json:"client"`
	Status    string `json:"status"`
}

func (c *Client) GetSummary() (*Summary, error) {
	data, err := c.doRequest("GET", "/stats/summary", nil)
	if err != nil {
		return nil, err
	}
	var s Summary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) GetTopDomains(count int) ([]TopItem, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/stats/top_domains?count=%d", count), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		TopDomains []struct {
			Domain string `json:"domain"`
			Count  int    `json:"count"`
		} `json:"top_domains"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	items := make([]TopItem, len(result.TopDomains))
	for i, d := range result.TopDomains {
		items[i] = TopItem{Domain: d.Domain, Count: d.Count}
	}
	return items, nil
}

func (c *Client) GetTopBlocked(count int) ([]TopItem, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/stats/top_blocked?count=%d", count), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		TopBlocked []struct {
			Domain string `json:"domain"`
			Count  int    `json:"count"`
		} `json:"top_blocked"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	items := make([]TopItem, len(result.TopBlocked))
	for i, d := range result.TopBlocked {
		items[i] = TopItem{Domain: d.Domain, Count: d.Count}
	}
	return items, nil
}

func (c *Client) GetRecentQueries(count int) ([]QueryLogEntry, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/queries?length=%d", count), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Queries []QueryLogEntry `json:"queries"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.Queries, nil
}

func (c *Client) Enable() error {
	_, err := c.doRequest("POST", "/dns/blocking", strings.NewReader(`{"blocking":true}`))
	return err
}

func (c *Client) Disable(seconds int) error {
	body := `{"blocking":false}`
	if seconds > 0 {
		body = fmt.Sprintf(`{"blocking":false,"timer":%d}`, seconds)
	}
	_, err := c.doRequest("POST", "/dns/blocking", strings.NewReader(body))
	return err
}

func (c *Client) AddDenylist(domain string) error {
	body := fmt.Sprintf(`{"domain":"%s"}`, domain)
	_, err := c.doRequest("POST", "/domains/deny/exact", strings.NewReader(body))
	return err
}

func (c *Client) RemoveDenylist(domain string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/domains/deny/exact/%s", domain), nil)
	return err
}

func (c *Client) AddAllowlist(domain string) error {
	body := fmt.Sprintf(`{"domain":"%s"}`, domain)
	_, err := c.doRequest("POST", "/domains/allow/exact", strings.NewReader(body))
	return err
}

func (c *Client) RemoveAllowlist(domain string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/domains/allow/exact/%s", domain), nil)
	return err
}

func (c *Client) GetDenylist() ([]string, error) {
	data, err := c.doRequest("GET", "/domains/deny/exact", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Domains []struct {
			Domain string `json:"domain"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	domains := make([]string, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = d.Domain
	}
	return domains, nil
}

func (c *Client) GetAllowlist() ([]string, error) {
	data, err := c.doRequest("GET", "/domains/allow/exact", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Domains []struct {
			Domain string `json:"domain"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	domains := make([]string, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = d.Domain
	}
	return domains, nil
}

func (c *Client) UpdateGravity() error {
	_, err := c.doRequest("POST", "/action/gravity", nil)
	return err
}

type DNSRecord struct {
	IP     string `json:"ip"`
	Domain string `json:"domain"`
}

func (c *Client) GetDNSRecords() ([]DNSRecord, error) {
	data, err := c.doRequest("GET", "/config/dns/hosts", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Hosts []string `json:"hosts"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		// try as direct array
		var hosts []string
		if err2 := json.Unmarshal(data, &hosts); err2 != nil {
			return nil, err
		}
		result.Hosts = hosts
	}
	records := make([]DNSRecord, 0, len(result.Hosts))
	for _, h := range result.Hosts {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 {
			records = append(records, DNSRecord{IP: parts[0], Domain: parts[1]})
		}
	}
	return records, nil
}

func (c *Client) AddDNSRecord(ip, domain string) error {
	body := fmt.Sprintf(`{"host":"%s %s"}`, ip, domain)
	_, err := c.doRequest("PUT", "/config/dns/hosts", strings.NewReader(body))
	return err
}

func (c *Client) RemoveDNSRecord(ip, domain string) error {
	body := fmt.Sprintf(`{"host":"%s %s"}`, ip, domain)
	_, err := c.doRequest("DELETE", "/config/dns/hosts", strings.NewReader(body))
	return err
}
