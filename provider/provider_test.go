package provider

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/protosio/protos/core"
	"github.com/protosio/protos/mock"
)

func TestProviderManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	amMock := mock.NewMockAppManager(ctrl)
	appMock := mock.NewMockApp(ctrl)
	rmMock := mock.NewMockResourceManager(ctrl)

	//
	// ProviderManager
	//

	// Testing provider manager creation
	dbMock.EXPECT().Register(gomock.Any()).Return().Times(1)
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1)
	pm := CreateManager(rmMock, amMock, dbMock)

	// If no app is registered as a provider for DNS, registration should be successful
	appMock.EXPECT().GetID().Return("appid01").Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	err := pm.Register(appMock, core.DNS)
	if err != nil {
		t.Error("pm.Register should not fail: ", err.Error())
	}
	// If the same app is registered as a provider for DNS, registration should fail
	appMock.EXPECT().GetID().Return("appid01").Times(2)
	err = pm.Register(appMock, core.DNS)
	if err == nil {
		t.Error("pm.Register should fail when provider already registered")
	}
	// If a different app is registered as a provider for DNS, registration should fail
	appMock.EXPECT().GetID().Return("appid02").Times(1)
	amMock.EXPECT().Read(gomock.Any()).Return(nil, nil).Times(1)
	err = pm.Register(appMock, core.DNS)
	if err == nil {
		t.Error("pm.Register should fail when provider already registered")
	}
	// If a non-existent app is registered as a provider for DNS, registration should continue
	appMock.EXPECT().GetID().Return("appid02").Times(1)
	amMock.EXPECT().Read(gomock.Any()).Return(nil, errors.New("")).Times(1)
	appMock.EXPECT().GetID().Return("appid01").Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	err = pm.Register(appMock, core.DNS)
	if err != nil {
		t.Error("pm.Register should not fail when non-existent app is registered as a provider ")
	}

	// Get
	appMock.EXPECT().GetID().Return("appid01").Times(1)
	prov, err := pm.Get(appMock)
	if err != nil {
		t.Error("Get should return the correct provider (DNS)")
	}

	appMock.EXPECT().GetID().Return("appid02").Times(1)
	appMock.EXPECT().GetName().Return("appname").Times(1)
	_, err = pm.Get(appMock)
	if err == nil {
		t.Error("Get should fail because the appID is different")
	}

	//
	// Provider
	//

	rscMock := mock.NewMockResource(ctrl)
	rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{"rscid01": rscMock}).Times(1)
	rscs := prov.GetResources()
	if len(rscs) != 1 {
		t.Errorf("The resource map should have 1 element but it has %d: %v", len(rscs), rscs)
	}
}
