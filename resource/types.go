package resource

const (
	Certificate = RType("certificate")
	DNS         = RType("dns")
	Mail        = RType("mail")
)

type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

type CertificateResource struct {
	Domains           []string `json:"domains"`
	PrivateKey        []byte   `json:"-"`
	Certificate       []byte   `json:"-"`
	IssuerCertificate []byte   `json:"-"`
}
