package resource

// RType is a string wrapper used for typechecking the resource types
type RType string

const (
	// Certificate represents a TLS/SSL certificate
	Certificate = RType("certificate")
	// DNS represents a DNS record
	DNS = RType("dns")
	// Mail is not used yet
	Mail = RType("mail")
)

// Type is an interface that satisfies all the resource types
type Type interface {
	Update(Type)
	Sanitize() Type
}

// DNSResource represents a DNS resource
type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

// Update method is not used for the DNS resource type
func (rsc *DNSResource) Update(newValue Type) {
}

// Sanitize removes any sensitive information from the resource
func (rsc *DNSResource) Sanitize() Type {
	return rsc
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
func (rsc *CertificateResource) Update(newValue Type) {
	newCert := newValue.(*CertificateResource)
	rsc.PrivateKey = newCert.PrivateKey
	rsc.Certificate = newCert.Certificate
	rsc.IssuerCertificate = newCert.IssuerCertificate
	rsc.CSR = newCert.CSR
}

// Sanitize removes any sensitive information from the resource
func (rsc *CertificateResource) Sanitize() Type {
	output := *rsc
	output.PrivateKey = []byte{}
	output.CSR = []byte{}
	return &output
}
