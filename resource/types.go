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
	Update(Type)
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

func (rsc *DNSResource) Update(newValue Type) {
}

// CertificateResource represents an SSL certificate resource
type CertificateResource struct {
	Domains           []string
	PrivateKey        []byte `hash:"-"`
	Certificate       []byte `hash:"-"`
	IssuerCertificate []byte `hash:"-"`
	CSR               []byte `hash:"-"`
}

func (rsc *CertificateResource) GetHash() string {
	return ""
}

func (rsc *CertificateResource) Update(newValue Type) {
	newCert := newValue.(*CertificateResource)
	rsc.PrivateKey = newCert.PrivateKey
	rsc.Certificate = newCert.Certificate
	rsc.IssuerCertificate = newCert.IssuerCertificate
	rsc.CSR = newCert.CSR
}
