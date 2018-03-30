package capability

// ResourceProvider capabilities
var ResourceProvider = New("ResourceProvider")
var RegisterResourceProvider = New("RegisterResourceProvider")
var DeregisterResourceProvider = New("DeregisterResourceProvider")
var GetProviderResources = New("GetProviderResources")
var SetResourceStatus = New("SetResourceStatus")

// ResourceConsumer capabilities
var ResourceConsumer = New("ResourceConsumer")

// Information capabilities
var GetInformation = New("GetInformation")

// User capabilities
var UserAdmin = New("UserAdmin")
var AuthUser = New("AuthUser")

// PublicDNS capability tells the platform to create a public dns record using the applications name
var PublicDNS = New("PublicDNS")

func createTree(root *Capability) {
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
