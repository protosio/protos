package resource

import "github.com/protosio/protos/internal/core"

// DNSResource represents a DNS resource
type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

// Update method is not used for the DNS resource type
func (rsc *DNSResource) Update(newValue core.ResourceValue) {
}

// Sanitize removes any sensitive information from the resource
func (rsc *DNSResource) Sanitize() core.ResourceValue {
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
func (rsc *CertificateResource) Update(newValue core.ResourceValue) {
	newCert := newValue.(*CertificateResource)
	rsc.PrivateKey = newCert.PrivateKey
	rsc.Certificate = newCert.Certificate
	rsc.IssuerCertificate = newCert.IssuerCertificate
	rsc.CSR = newCert.CSR
}

// Sanitize removes any sensitive information from the resource
func (rsc *CertificateResource) Sanitize() core.ResourceValue {
	output := *rsc
	output.PrivateKey = []byte{}
	output.CSR = []byte{}
	return &output
}

// GetCertificate returns the resource certificate
func (rsc *CertificateResource) GetCertificate() []byte {
	return rsc.Certificate
}

// GetPrivateKey returns the private key of the certificate
func (rsc *CertificateResource) GetPrivateKey() []byte {
	return rsc.PrivateKey
}
