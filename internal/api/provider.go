package api

type Provider interface {
	GetSummary() (*Summary, error)
	GetTopDomains(count int) ([]TopItem, error)
	GetTopBlocked(count int) ([]TopItem, error)
	GetRecentQueries(count int) ([]QueryLogEntry, error)
	GetDenylist() ([]string, error)
	GetAllowlist() ([]string, error)
	AddDenylist(domain string) error
	RemoveDenylist(domain string) error
	AddAllowlist(domain string) error
	RemoveAllowlist(domain string) error
	GetDNSRecords() ([]DNSRecord, error)
	AddDNSRecord(ip, domain string) error
	RemoveDNSRecord(ip, domain string) error
	Enable() error
	Disable(seconds int) error
	UpdateGravity() error
	Name() string
}
