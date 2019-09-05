package resource

import "protos/internal/core"

const (
	// Certificate represents a TLS/SSL certificate
	Certificate = core.RType("certificate")
	// DNS represents a DNS record
	DNS = core.RType("dns")
	// Mail is not used yet
	Mail = core.RType("mail")
)

// DNSResource represents a DNS resource
type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

// Update method is not used for the DNS resource type
func (rsc *DNSResource) Update(newValue core.Type) {
}

// Sanitize removes any sensitive information from the resource
func (rsc *DNSResource) Sanitize() core.Type {
	return rsc
}

// IsType is used to check if the DNS resource is of a specific type
func (rsc *DNSResource) IsType(t string) bool {
	return rsc.Type == t
}

// GetName returns the host field of the DNS record
func (rsc *DNSResource) GetName() string {
	return rsc.Host
}

// GetValue returns the value field of the DNS record
func (rsc *DNSResource) GetValue() string {
	return rsc.Value
}

// CertificateResource represents an SSL certificate resource
type CertificateResource struct {
	Domains           []string
	PrivateKey        []byte `json:"privatekey,omitempty" hash:"-"`
	Certificate       []byte `json:"certificate,omitempty" hash:"-"`
	IssuerCertificate []byte `json:"issuercertificate,omitempty" hash:"-"`
	CSR               []byte `json:"csr,omitempty" hash:"-"`
}

// Update takes a new resource type value and updates the relevant fields
func (rsc *CertificateResource) Update(newValue core.Type) {
	newCert := newValue.(*CertificateResource)
	rsc.PrivateKey = newCert.PrivateKey
	rsc.Certificate = newCert.Certificate
	rsc.IssuerCertificate = newCert.IssuerCertificate
	rsc.CSR = newCert.CSR
}

// Sanitize removes any sensitive information from the resource
func (rsc *CertificateResource) Sanitize() core.Type {
	output := *rsc
	output.PrivateKey = []byte{}
	output.CSR = []byte{}
	return &output
}
