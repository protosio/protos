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
	Sanitize() Type
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

func (rsc *CertificateResource) Sanitize() Type {
	output := *rsc
	output.PrivateKey = []byte{}
	output.Certificate = []byte{}
	output.IssuerCertificate = []byte{}
	output.CSR = []byte{}
	return &output
}
