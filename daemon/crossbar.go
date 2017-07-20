package daemon

// Provider defines a Protos resource provider
type Provider struct {
	resource Resource
}

// Consumer defines a Protos resource consumer
type Consumer struct {
	resource Resource
}

// RegisterProvider registers a resource provider
func RegisterProvider(resource Resource, appID string) error {
	return nil
}
