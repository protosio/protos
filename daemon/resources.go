package daemon

// DNS represents a dns record resource
type DNS struct {
	Type  string
	Host  string
	Value string
	TTL   int
}

// Resource defines a Protos resource
type Resource struct {
	Type string
}
