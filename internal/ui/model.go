package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rodrigodealer/pihole-tui/internal/api"
)

type view int

const (
	viewDashboard view = iota
	viewLive
	viewTopDomains
	viewTopBlocked
	viewQueryLog
	viewDenylist
	viewAllowlist
	viewDNSRecords
	viewAddDomain
	viewAddDNS
)

type Model struct {
	client     *api.Client
	view       view
	cursor     int
	summary    *api.Summary
	topItems   []api.TopItem
	queries    []api.QueryLogEntry
	domains    []string
	dnsRecords []api.DNSRecord
	textInput  textinput.Model
	textInput2 textinput.Model
	addTarget  string // "deny", "allow", or "dns"
	dnsStep    int    // 0=IP, 1=domain
	message    string
	err        error
	width      int
	height     int
	loading    bool
	liveActive bool
}

type summaryMsg *api.Summary
type topDomainsMsg []api.TopItem
type topBlockedMsg []api.TopItem
type queriesMsg []api.QueryLogEntry
type domainsMsg []string
type dnsRecordsMsg []api.DNSRecord
type actionMsg string
type errMsg error
type tickMsg time.Time

func NewModel(client *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.CharLimit = 253
	ti.Width = 40

	ti2 := textinput.New()
	ti2.Placeholder = "hostname.local"
	ti2.CharLimit = 253
	ti2.Width = 40

	return Model{
		client:     client,
		view:       viewDashboard,
		textInput:  ti,
		textInput2: ti2,
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchSummary()
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetchSummary() tea.Cmd {
	return func() tea.Msg {
		s, err := m.client.GetSummary()
		if err != nil {
			return errMsg(err)
		}
		return summaryMsg(s)
	}
}

func (m Model) fetchTopDomains() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.GetTopDomains(15)
		if err != nil {
			return errMsg(err)
		}
		return topDomainsMsg(items)
	}
}

func (m Model) fetchTopBlocked() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.GetTopBlocked(15)
		if err != nil {
			return errMsg(err)
		}
		return topBlockedMsg(items)
	}
}

func (m Model) fetchQueries() tea.Cmd {
	return func() tea.Msg {
		q, err := m.client.GetRecentQueries(20)
		if err != nil {
			return errMsg(err)
		}
		return queriesMsg(q)
	}
}

func (m Model) fetchDenylist() tea.Cmd {
	return func() tea.Msg {
		d, err := m.client.GetDenylist()
		if err != nil {
			return errMsg(err)
		}
		return domainsMsg(d)
	}
}

func (m Model) fetchAllowlist() tea.Cmd {
	return func() tea.Msg {
		d, err := m.client.GetAllowlist()
		if err != nil {
			return errMsg(err)
		}
		return domainsMsg(d)
	}
}

func (m Model) fetchDNSRecords() tea.Cmd {
	return func() tea.Msg {
		r, err := m.client.GetDNSRecords()
		if err != nil {
			return errMsg(err)
		}
		return dnsRecordsMsg(r)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		if m.liveActive {
			return m, tea.Batch(m.fetchSummary(), tickEvery(2*time.Second))
		}
		return m, nil

	case summaryMsg:
		m.summary = msg
		m.loading = false
		m.err = nil
		return m, nil

	case topDomainsMsg:
		m.topItems = msg
		m.loading = false
		return m, nil

	case topBlockedMsg:
		m.topItems = msg
		m.loading = false
		return m, nil

	case queriesMsg:
		m.queries = msg
		m.loading = false
		return m, nil

	case domainsMsg:
		m.domains = msg
		m.loading = false
		return m, nil

	case dnsRecordsMsg:
		m.dnsRecords = msg
		m.loading = false
		return m, nil

	case actionMsg:
		m.message = string(msg)
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil
	}

	if m.view == viewAddDomain {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	if m.view == viewAddDNS {
		var cmd tea.Cmd
		if m.dnsStep == 0 {
			m.textInput, cmd = m.textInput.Update(msg)
		} else {
			m.textInput2, cmd = m.textInput2.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.view == viewAddDomain {
		return m.handleAddDomainKey(msg)
	}
	if m.view == viewAddDNS {
		return m.handleAddDNSKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.liveActive = false
		return m, tea.Quit

	case "esc":
		if m.view == viewLive {
			m.liveActive = false
			m.view = viewDashboard
			m.cursor = 0
			return m, nil
		}
		if m.view != viewDashboard {
			m.view = viewDashboard
			m.cursor = 0
			m.message = ""
			m.err = nil
			return m, m.fetchSummary()
		}
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		m.cursor++
		return m, nil

	case "r":
		m.loading = true
		return m, m.refreshCurrent()

	case "a":
		if m.view == viewDNSRecords {
			m.view = viewAddDNS
			m.dnsStep = 0
			m.textInput.Placeholder = "192.168.0.100"
			m.textInput.Focus()
			m.textInput.Reset()
			m.textInput2.Reset()
			return m, nil
		}
		if m.view == viewDenylist {
			m.view = viewAddDomain
			m.addTarget = "deny"
			m.textInput.Placeholder = "example.com"
			m.textInput.Focus()
			m.textInput.Reset()
			return m, nil
		}
		if m.view == viewAllowlist {
			m.view = viewAddDomain
			m.addTarget = "allow"
			m.textInput.Placeholder = "example.com"
			m.textInput.Focus()
			m.textInput.Reset()
			return m, nil
		}
		return m, nil

	case "enter":
		if m.view == viewDashboard {
			return m.handleMenuSelect()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleAddDomainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.addTarget == "deny" {
			m.view = viewDenylist
		} else {
			m.view = viewAllowlist
		}
		m.textInput.Reset()
		return m, nil
	case "enter":
		domain := m.textInput.Value()
		if domain == "" {
			return m, nil
		}
		m.loading = true
		m.textInput.Reset()
		target := m.addTarget
		if target == "deny" {
			m.view = viewDenylist
		} else {
			m.view = viewAllowlist
		}
		return m, func() tea.Msg {
			var err error
			if target == "deny" {
				err = m.client.AddDenylist(domain)
			} else {
				err = m.client.AddAllowlist(domain)
			}
			if err != nil {
				return errMsg(err)
			}
			return actionMsg(fmt.Sprintf("Added %s to %slist", domain, target))
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleAddDNSKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewDNSRecords
		m.textInput.Reset()
		m.textInput2.Reset()
		return m, nil
	case "enter":
		if m.dnsStep == 0 {
			if m.textInput.Value() == "" {
				return m, nil
			}
			m.dnsStep = 1
			m.textInput2.Focus()
			return m, nil
		}
		ip := m.textInput.Value()
		domain := m.textInput2.Value()
		if domain == "" {
			return m, nil
		}
		m.loading = true
		m.textInput.Reset()
		m.textInput2.Reset()
		m.view = viewDNSRecords
		return m, func() tea.Msg {
			if err := m.client.AddDNSRecord(ip, domain); err != nil {
				return errMsg(err)
			}
			return actionMsg(fmt.Sprintf("Added %s -> %s", domain, ip))
		}
	}
	var cmd tea.Cmd
	if m.dnsStep == 0 {
		m.textInput, cmd = m.textInput.Update(msg)
	} else {
		m.textInput2, cmd = m.textInput2.Update(msg)
	}
	return m, cmd
}

func (m Model) handleMenuSelect() (tea.Model, tea.Cmd) {
	m.loading = true
	m.err = nil
	m.message = ""
	switch m.cursor {
	case 0: // Dashboard (refresh)
		return m, m.fetchSummary()
	case 1: // Live Dashboard
		m.view = viewLive
		m.liveActive = true
		m.cursor = 0
		return m, tea.Batch(m.fetchSummary(), tickEvery(2*time.Second))
	case 2: // Top Domains
		m.view = viewTopDomains
		m.cursor = 0
		return m, m.fetchTopDomains()
	case 3: // Top Blocked
		m.view = viewTopBlocked
		m.cursor = 0
		return m, m.fetchTopBlocked()
	case 4: // Query Log
		m.view = viewQueryLog
		m.cursor = 0
		return m, m.fetchQueries()
	case 5: // DNS Records
		m.view = viewDNSRecords
		m.cursor = 0
		return m, m.fetchDNSRecords()
	case 6: // Denylist
		m.view = viewDenylist
		m.cursor = 0
		return m, m.fetchDenylist()
	case 7: // Allowlist
		m.view = viewAllowlist
		m.cursor = 0
		return m, m.fetchAllowlist()
	case 8: // Enable
		return m, func() tea.Msg {
			if err := m.client.Enable(); err != nil {
				return errMsg(err)
			}
			return actionMsg("Blocking enabled")
		}
	case 9: // Disable (30s)
		return m, func() tea.Msg {
			if err := m.client.Disable(30); err != nil {
				return errMsg(err)
			}
			return actionMsg("Blocking disabled for 30s")
		}
	case 10: // Disable (5m)
		return m, func() tea.Msg {
			if err := m.client.Disable(300); err != nil {
				return errMsg(err)
			}
			return actionMsg("Blocking disabled for 5m")
		}
	case 11: // Disable (indefinitely)
		return m, func() tea.Msg {
			if err := m.client.Disable(0); err != nil {
				return errMsg(err)
			}
			return actionMsg("Blocking disabled")
		}
	case 12: // Update Gravity
		return m, func() tea.Msg {
			if err := m.client.UpdateGravity(); err != nil {
				return errMsg(err)
			}
			return actionMsg("Gravity update started")
		}
	}
	return m, nil
}

func (m Model) refreshCurrent() tea.Cmd {
	switch m.view {
	case viewDashboard, viewLive:
		return m.fetchSummary()
	case viewTopDomains:
		return m.fetchTopDomains()
	case viewTopBlocked:
		return m.fetchTopBlocked()
	case viewQueryLog:
		return m.fetchQueries()
	case viewDenylist:
		return m.fetchDenylist()
	case viewAllowlist:
		return m.fetchAllowlist()
	case viewDNSRecords:
		return m.fetchDNSRecords()
	}
	return nil
}

func (m Model) View() string {
	switch m.view {
	case viewDashboard:
		return m.viewDashboard()
	case viewLive:
		return m.viewLiveDashboard()
	case viewTopDomains:
		return m.viewTopItems("Top Domains")
	case viewTopBlocked:
		return m.viewTopItems("Top Blocked")
	case viewQueryLog:
		return m.viewQueryLog()
	case viewDenylist:
		return m.viewDomainList("Denylist", "deny")
	case viewAllowlist:
		return m.viewDomainList("Allowlist", "allow")
	case viewDNSRecords:
		return m.viewDNSRecords()
	case viewAddDomain:
		return m.viewAddDomain()
	case viewAddDNS:
		return m.viewAddDNSRecord()
	}
	return ""
}

func (m Model) viewDashboard() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Pi-hole Remote Control") + "\n\n")

	if m.summary != nil {
		b.WriteString(m.renderSummaryBox() + "\n\n")
	}

	menuItems := []string{
		"Refresh Dashboard",
		"Live Dashboard",
		"Top Domains",
		"Top Blocked",
		"Query Log",
		"DNS Records",
		"Denylist",
		"Allowlist",
		"Enable Blocking",
		"Disable 30s",
		"Disable 5m",
		"Disable Indefinitely",
		"Update Gravity",
	}

	maxCursor := len(menuItems) - 1
	if m.cursor > maxCursor {
		m.cursor = maxCursor
	}

	for i, item := range menuItems {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("▸ "+item) + "\n")
		} else {
			b.WriteString(menuItemStyle.Render("  "+item) + "\n")
		}
	}

	if m.message != "" {
		b.WriteString("\n" + successStyle.Render("  ✓ "+m.message) + "\n")
	}
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}
	if m.loading {
		b.WriteString("\n" + statLabelStyle.Render("  Loading...") + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • enter select • r refresh • q quit"))

	return b.String()
}

func (m Model) viewLiveDashboard() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Pi-hole Live Dashboard") + "  ")
	b.WriteString(statusEnabledStyle.Render("● LIVE") + "\n\n")

	if m.summary != nil {
		b.WriteString(m.renderSummaryBox() + "\n\n")

		qps := float64(m.summary.Queries) / 86400.0 * 100
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("~Queries/min:"),
			statValueStyle.Render(fmt.Sprintf("%.1f", qps))))
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("Forwarded:"),
			statValueStyle.Render(fmt.Sprintf("%d", m.summary.QueriesForwarded))))
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("Cached:"),
			statValueStyle.Render(fmt.Sprintf("%d", m.summary.QueriesCached))))
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("Unique Domains:"),
			statValueStyle.Render(fmt.Sprintf("%d", m.summary.UniqueDomains))))
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("Refreshes every 2s • esc back • q quit"))
	return b.String()
}

func (m Model) renderSummaryBox() string {
	status := statusEnabledStyle.Render("● Enabled")
	if m.summary.Status != "enabled" {
		status = statusDisabledStyle.Render("● Disabled")
	}

	pct := fmt.Sprintf("%.1f%%", m.summary.BlockedPct)
	bar := renderBar(m.summary.BlockedPct, 30)

	stats := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("  %s  %s", statLabelStyle.Render("Status:"), status),
		"",
		fmt.Sprintf("  %s  %s", statLabelStyle.Render("Total Queries:"), statValueStyle.Render(fmt.Sprintf("%d", m.summary.Queries))),
		fmt.Sprintf("  %s  %s  %s", statLabelStyle.Render("Blocked:"), statValueStyle.Render(fmt.Sprintf("%d", m.summary.Blocked)), barFilledStyle.Render(pct)),
		fmt.Sprintf("  %s  %s", statLabelStyle.Render("Block Rate:"), bar),
		fmt.Sprintf("  %s  %s", statLabelStyle.Render("Domains on List:"), statValueStyle.Render(fmt.Sprintf("%d", m.summary.DomainsOnList))),
		fmt.Sprintf("  %s  %s", statLabelStyle.Render("Clients:"), statValueStyle.Render(fmt.Sprintf("%d", m.summary.ClientsEver))),
	)
	return boxStyle.Render(stats)
}

func (m Model) viewTopItems(title string) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(title) + "\n")

	if len(m.topItems) == 0 {
		b.WriteString(statLabelStyle.Render("  No data available") + "\n")
	} else {
		maxCount := 0
		for _, item := range m.topItems {
			if item.Count > maxCount {
				maxCount = item.Count
			}
		}
		for _, item := range m.topItems {
			bar := ""
			if maxCount > 0 {
				width := int(float64(item.Count) / float64(maxCount) * 20)
				bar = barFilledStyle.Render(strings.Repeat("█", width)) + barEmptyStyle.Render(strings.Repeat("░", 20-width))
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				statValueStyle.Render(fmt.Sprintf("%6d", item.Count)),
				bar,
				item.Domain))
		}
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("r refresh • esc back • q quit"))
	return b.String()
}

func (m Model) viewQueryLog() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Recent Queries") + "\n")

	if len(m.queries) == 0 {
		b.WriteString(statLabelStyle.Render("  No queries") + "\n")
	} else {
		for _, q := range m.queries {
			status := statusEnabledStyle.Render("OK")
			if q.Status == "blocked" || q.Status == "BLOCKED" {
				status = statusDisabledStyle.Render("BL")
			}
			b.WriteString(fmt.Sprintf("  %s  %-6s  %-40s  %s\n",
				status,
				q.Type,
				truncate(q.Domain, 40),
				q.Client))
		}
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("r refresh • esc back • q quit"))
	return b.String()
}

func (m Model) viewDNSRecords() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Local DNS Records") + "\n")

	if len(m.dnsRecords) == 0 {
		b.WriteString(statLabelStyle.Render("  No records") + "\n")
	} else {
		for _, r := range m.dnsRecords {
			b.WriteString(fmt.Sprintf("  %-16s  %s\n",
				statValueStyle.Render(r.IP),
				r.Domain))
		}
	}

	if m.message != "" {
		b.WriteString("\n" + successStyle.Render("  ✓ "+m.message) + "\n")
	}
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("a add • r refresh • esc back • q quit"))
	return b.String()
}

func (m Model) viewDomainList(title, listType string) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(title) + "\n")

	if len(m.domains) == 0 {
		b.WriteString(statLabelStyle.Render("  Empty") + "\n")
	} else {
		for _, d := range m.domains {
			b.WriteString(fmt.Sprintf("  • %s\n", d))
		}
	}

	if m.message != "" {
		b.WriteString("\n" + successStyle.Render("  ✓ "+m.message) + "\n")
	}
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("  ✗ "+m.err.Error()) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("a add • r refresh • esc back • q quit"))
	return b.String()
}

func (m Model) viewAddDomain() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("Add to %slist", m.addTarget)) + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("enter confirm • esc cancel"))
	return b.String()
}

func (m Model) viewAddDNSRecord() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add DNS Record") + "\n\n")

	if m.dnsStep == 0 {
		b.WriteString(statLabelStyle.Render("  IP Address:") + "\n")
		b.WriteString("  " + m.textInput.View() + "\n\n")
	} else {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("IP Address:"),
			statValueStyle.Render(m.textInput.Value())))
		b.WriteString(statLabelStyle.Render("  Hostname:") + "\n")
		b.WriteString("  " + m.textInput2.View() + "\n\n")
	}

	b.WriteString(helpStyle.Render("enter confirm • esc cancel"))
	return b.String()
}

func renderBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	return barFilledStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", width-filled))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
