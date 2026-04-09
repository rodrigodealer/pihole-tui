package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AdGuardClient struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

func NewAdGuardClient(baseURL, username, password string) *AdGuardClient {
	return &AdGuardClient{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AdGuardClient) Name() string { return "adguard" }

func (c *AdGuardClient) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.BaseURL+"/control"+endpoint, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
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
		return nil, fmt.Errorf("AdGuard API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *AdGuardClient) GetSummary() (*Summary, error) {
	data, err := c.doRequest("GET", "/stats", nil)
	if err != nil {
		return nil, err
	}
	var stats struct {
		NumDNSQueries       int     `json:"num_dns_queries"`
		NumBlockedFiltering int     `json:"num_blocked_filtering"`
		NumReplacedSafesearch int   `json:"num_replaced_safesearch"`
		AvgProcessingTime   float64 `json:"avg_processing_time"`
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}

	statusData, err := c.doRequest("GET", "/status", nil)
	if err != nil {
		return nil, err
	}
	var status struct {
		ProtectionEnabled bool `json:"protection_enabled"`
		Running           bool `json:"running"`
	}
	if err := json.Unmarshal(statusData, &status); err != nil {
		return nil, err
	}

	pct := 0.0
	if stats.NumDNSQueries > 0 {
		pct = float64(stats.NumBlockedFiltering) / float64(stats.NumDNSQueries) * 100
	}

	statusStr := "disabled"
	if status.ProtectionEnabled {
		statusStr = "enabled"
	}

	return &Summary{
		Queries:    stats.NumDNSQueries,
		Blocked:    stats.NumBlockedFiltering,
		BlockedPct: pct,
		Status:     statusStr,
	}, nil
}

func (c *AdGuardClient) GetTopDomains(count int) ([]TopItem, error) {
	data, err := c.doRequest("GET", "/stats", nil)
	if err != nil {
		return nil, err
	}
	var stats struct {
		TopQueriedDomains []map[string]int `json:"top_queried_domains"`
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return mapToTopItems(stats.TopQueriedDomains, count), nil
}

func (c *AdGuardClient) GetTopBlocked(count int) ([]TopItem, error) {
	data, err := c.doRequest("GET", "/stats", nil)
	if err != nil {
		return nil, err
	}
	var stats struct {
		TopBlockedDomains []map[string]int `json:"top_blocked_domains"`
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return mapToTopItems(stats.TopBlockedDomains, count), nil
}

func (c *AdGuardClient) GetRecentQueries(count int) ([]QueryLogEntry, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/querylog?limit=%d", count), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []struct {
			Question struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"question"`
			Client string `json:"client"`
			Reason string `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	entries := make([]QueryLogEntry, len(result.Data))
	for i, q := range result.Data {
		status := "OK"
		if q.Reason != "" && q.Reason != "NotFilteredNotFound" && q.Reason != "NotFilteredWhiteList" {
			status = "blocked"
		}
		entries[i] = QueryLogEntry{
			Domain: strings.TrimSuffix(q.Question.Name, "."),
			Type:   q.Question.Type,
			Client: q.Client,
			Status: status,
		}
	}
	return entries, nil
}

func (c *AdGuardClient) GetDenylist() ([]string, error) {
	return c.getFilteringRules("blocked_services")
}

func (c *AdGuardClient) GetAllowlist() ([]string, error) {
	data, err := c.doRequest("GET", "/filtering/status", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		UserRules []string `json:"user_rules"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	var allowed []string
	for _, r := range result.UserRules {
		if strings.HasPrefix(r, "@@||") {
			domain := strings.TrimPrefix(r, "@@||")
			domain = strings.TrimSuffix(domain, "^")
			allowed = append(allowed, domain)
		}
	}
	return allowed, nil
}

func (c *AdGuardClient) AddDenylist(domain string) error {
	rules, err := c.getUserRules()
	if err != nil {
		return err
	}
	rule := "||" + domain + "^"
	rules = append(rules, rule)
	return c.setUserRules(rules)
}

func (c *AdGuardClient) RemoveDenylist(domain string) error {
	rules, err := c.getUserRules()
	if err != nil {
		return err
	}
	rule := "||" + domain + "^"
	var filtered []string
	for _, r := range rules {
		if r != rule {
			filtered = append(filtered, r)
		}
	}
	return c.setUserRules(filtered)
}

func (c *AdGuardClient) AddAllowlist(domain string) error {
	rules, err := c.getUserRules()
	if err != nil {
		return err
	}
	rule := "@@||" + domain + "^"
	rules = append(rules, rule)
	return c.setUserRules(rules)
}

func (c *AdGuardClient) RemoveAllowlist(domain string) error {
	rules, err := c.getUserRules()
	if err != nil {
		return err
	}
	rule := "@@||" + domain + "^"
	var filtered []string
	for _, r := range rules {
		if r != rule {
			filtered = append(filtered, r)
		}
	}
	return c.setUserRules(filtered)
}

func (c *AdGuardClient) GetDNSRecords() ([]DNSRecord, error) {
	data, err := c.doRequest("GET", "/rewrite/list", nil)
	if err != nil {
		return nil, err
	}
	var rewrites []struct {
		Domain string `json:"domain"`
		Answer string `json:"answer"`
	}
	if err := json.Unmarshal(data, &rewrites); err != nil {
		return nil, err
	}
	records := make([]DNSRecord, len(rewrites))
	for i, r := range rewrites {
		records[i] = DNSRecord{IP: r.Answer, Domain: r.Domain}
	}
	return records, nil
}

func (c *AdGuardClient) AddDNSRecord(ip, domain string) error {
	body := fmt.Sprintf(`{"domain":"%s","answer":"%s"}`, domain, ip)
	_, err := c.doRequest("POST", "/rewrite/add", strings.NewReader(body))
	return err
}

func (c *AdGuardClient) RemoveDNSRecord(ip, domain string) error {
	body := fmt.Sprintf(`{"domain":"%s","answer":"%s"}`, domain, ip)
	_, err := c.doRequest("POST", "/rewrite/delete", strings.NewReader(body))
	return err
}

func (c *AdGuardClient) Enable() error {
	_, err := c.doRequest("POST", "/dns_config", strings.NewReader(`{"protection_enabled":true}`))
	return err
}

func (c *AdGuardClient) Disable(seconds int) error {
	body := `{"protection_enabled":false}`
	if seconds > 0 {
		body = fmt.Sprintf(`{"protection_enabled":false,"protection_disabled_until":"%s"}`,
			time.Now().Add(time.Duration(seconds)*time.Second).Format(time.RFC3339))
	}
	_, err := c.doRequest("POST", "/dns_config", strings.NewReader(body))
	return err
}

func (c *AdGuardClient) UpdateGravity() error {
	_, err := c.doRequest("POST", "/filtering/refresh", strings.NewReader(`{"whitelist":false}`))
	return err
}

func (c *AdGuardClient) getUserRules() ([]string, error) {
	data, err := c.doRequest("GET", "/filtering/status", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		UserRules []string `json:"user_rules"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.UserRules, nil
}

func (c *AdGuardClient) setUserRules(rules []string) error {
	body := fmt.Sprintf(`{"rules":%s}`, mustJSON(rules))
	_, err := c.doRequest("POST", "/filtering/set_rules", strings.NewReader(body))
	return err
}

func (c *AdGuardClient) getFilteringRules(_ string) ([]string, error) {
	rules, err := c.getUserRules()
	if err != nil {
		return nil, err
	}
	var blocked []string
	for _, r := range rules {
		if strings.HasPrefix(r, "||") && !strings.HasPrefix(r, "@@") {
			domain := strings.TrimPrefix(r, "||")
			domain = strings.TrimSuffix(domain, "^")
			blocked = append(blocked, domain)
		}
	}
	return blocked, nil
}

func mapToTopItems(items []map[string]int, count int) []TopItem {
	var result []TopItem
	for _, m := range items {
		for domain, cnt := range m {
			result = append(result, TopItem{Domain: domain, Count: cnt})
		}
		if len(result) >= count {
			break
		}
	}
	if len(result) > count {
		result = result[:count]
	}
	return result
}

func mustJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
