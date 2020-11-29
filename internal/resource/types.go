package resource

import (
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

// ResourceType is a string wrapper used for typechecking the resource types
type ResourceType string

const (
	// Certificate represents a TLS/SSL certificate
	Certificate = ResourceType("certificate")
	// DNS represents a DNS record
	DNS = ResourceType("dns")
	// Mail is not used yet
	Mail = ResourceType("mail")
)

// DNSResource represents a DNS resource
type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

// Update method is not used for the DNS resource type
func (rsc *DNSResource) Update(newValue ResourceValue) {
	newDNS := newValue.(*DNSResource)
	rsc.Value = newDNS.Value
	rsc.TTL = newDNS.TTL
}

// UpdateValueAndTTL updates the value and TTL of the dns record
func (rsc *DNSResource) UpdateValueAndTTL(value string, ttl int) {
	rsc.Value = value
	rsc.TTL = ttl
}

// Sanitize removes any sensitive information from the resource
func (rsc *DNSResource) Sanitize() ResourceValue {
	return rsc
}

// MarshalNoms encodes the resource into a noms value type.
func (rsc *DNSResource) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	return marshal.Marshal(vrw, *rsc)
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
func (rsc *CertificateResource) Update(newValue ResourceValue) {
	newCert := newValue.(*CertificateResource)
	rsc.PrivateKey = newCert.PrivateKey
	rsc.Certificate = newCert.Certificate
	rsc.IssuerCertificate = newCert.IssuerCertificate
	rsc.CSR = newCert.CSR
}

// Sanitize removes any sensitive information from the resource
func (rsc *CertificateResource) Sanitize() ResourceValue {
	output := *rsc
	output.PrivateKey = []byte{}
	output.CSR = []byte{}
	return &output
}

// MarshalNoms encodes the resource into a noms value type
func (rsc *CertificateResource) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	return marshal.Marshal(vrw, *rsc)
}

// GetCertificate returns the resource certificate
func (rsc *CertificateResource) GetCertificate() []byte {
	return rsc.Certificate
}

// GetPrivateKey returns the private key of the certificate
func (rsc *CertificateResource) GetPrivateKey() []byte {
	return rsc.PrivateKey
}
