package resource

type RType string

const (
	Certificate = RType("certificate")
	DNS         = RType("dns")
	Mail        = RType("mail")
)

// Type is an interface that satisfies all the resource types
type Type interface {
	GetHash() string
}

// DNSResource represents a DNS resource
type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

func (rsc *DNSResource) GetHash() string {
	return ""
}

// CertificateResource represents an SSL certificate resource
type CertificateResource struct {
	Domains           []string `json:"domains"`
	PrivateKey        []byte   `json:"-" hash:"-"`
	Certificate       []byte   `json:"-" hash:"-"`
	IssuerCertificate []byte   `json:"-" hash:"-"`
}

func (rsc *CertificateResource) GetHash() string {
	return ""
}
