package capability

// ResourceProvider capabilities
var ResourceProvider = New("ResourceProvider")
var RegisterResourceProvider = New("RegisterResourceProvider")
var DeregisterResourceProvider = New("DeregisterResourceProvider")
var GetProviderResources = New("GetProviderResources")
var SetResourceStatus = New("SetResourceStatus")

func createTree(root *Capability) {
	RegisterResourceProvider.SetParent(ResourceProvider)
	DeregisterResourceProvider.SetParent(ResourceProvider)
	GetProviderResources.SetParent(ResourceProvider)
	SetResourceStatus.SetParent(ResourceProvider)

	ResourceProvider.SetParent(root)
}
