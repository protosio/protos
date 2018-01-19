package api

import (
	"protos/capability"
)

func createCapabilities() {

	var capResourceProvider = capability.New("ResourceProvider")
	var capRegisterResourceProvider = capability.New("RegisterResourceProvider")
	var capDeregisterResourceProvider = capability.New("DeregisterResourceProvider")
	var capGetProviderResources = capability.New("GetProviderResources")
	var capSetResourceStatus = capability.New("SetResourceStatus")

	capRegisterResourceProvider.SetParent(capResourceProvider)
	capDeregisterResourceProvider.SetParent(capResourceProvider)
	capGetProviderResources.SetParent(capResourceProvider)
	capSetResourceStatus.SetParent(capResourceProvider)

	capResourceProvider.SetParent(&capability.RC)

	// Match capabilities to their respective methods
	capability.SetMethodCap(CregisterResourceProvider, capRegisterResourceProvider)

}
