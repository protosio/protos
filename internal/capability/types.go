package capability

func (cm *Manager) createTree(root *Capability) {
	// ResourceProvider capabilities
	ResourceProvider := cm.New("ResourceProvider")
	RegisterResourceProvider := cm.New("RegisterResourceProvider")
	DeregisterResourceProvider := cm.New("DeregisterResourceProvider")
	GetProviderResources := cm.New("GetProviderResources")
	SetResourceStatus := cm.New("SetResourceStatus")

	// ResourceConsumer capabilities
	ResourceConsumer := cm.New("ResourceConsumer")

	// Information capabilities
	GetInformation := cm.New("GetInformation")

	// User capabilities
	UserAdmin := cm.New("UserAdmin")
	AuthUser := cm.New("AuthUser")

	// PublicDNS capability tells the platform to create a public dns record using the applications name
	PublicDNS := cm.New("PublicDNS")

	RegisterResourceProvider.SetParent(ResourceProvider)
	DeregisterResourceProvider.SetParent(ResourceProvider)
	GetProviderResources.SetParent(ResourceProvider)
	SetResourceStatus.SetParent(ResourceProvider)
	ResourceConsumer.SetParent(ResourceProvider)

	ResourceProvider.SetParent(root)
	GetInformation.SetParent(root)

	UserAdmin.SetParent(root)
	AuthUser.SetParent(root)

	PublicDNS.SetParent(root)
}
