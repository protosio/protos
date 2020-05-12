package provider

import (
	"errors"
	"testing"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/mock"

	"github.com/golang/mock/gomock"
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
	dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			providers := to.(*[]Provider)
			*providers = append(*providers, Provider{Type: core.DNS})
		})
	pm := CreateManager(rmMock, amMock, dbMock)

	// If no app is registered as a provider for DNS, registration should be successful
	appMock.EXPECT().GetID().Return("appid01").Times(1)
	dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)
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
	dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	err = pm.Register(appMock, core.DNS)
	if err != nil {
		t.Error("pm.Register should not fail when non-existent app is registered as a provider ")
	}

	// Get
	appMock.EXPECT().GetID().Return("appid01").Times(1)
	prov, err := pm.Get(appMock)
	if err != nil {
		t.Error("Get() should return the correct provider (DNS)")
	}
	if prov.TypeName() != "dns" {
		t.Errorf("TypeName() should return the correct provider type: dns vs %s", prov.TypeName())
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

	// ToDo: this test should look in details at the filter function passed to select
	rscMock := mock.NewMockResource(ctrl)
	rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{"rscid01": rscMock}).Times(1)
	rscs := prov.GetResources()
	if len(rscs) != 1 {
		t.Errorf("The resource map should have 1 element but it has %d: %v", len(rscs), rscs)
	}

	rmMock.EXPECT().Get("rscid01").Return(rscMock, nil).Times(1)
	rscMock.EXPECT().GetType().Return(core.DNS).Times(1)
	rsc := prov.GetResource("rscid01")
	if rsc != rscMock {
		t.Error("Resource rscid01(mock) should be retrievable from the DNS provider")
	}

	rmMock.EXPECT().Get("rscid02").Return(rscMock, errors.New("Resource id not found")).Times(1)
	rsc = prov.GetResource("rscid02")
	if rsc != nil {
		t.Error("Wrong resource id should lead to the return of nil")
	}

	rmMock.EXPECT().Get("rscid01").Return(rscMock, nil).Times(1)
	rscMock.EXPECT().GetType().Return(core.Certificate).Times(2)
	rsc = prov.GetResource("rscid01")
	if rsc != nil {
		t.Error("Wrong resource type should lead to the return of nil")
	}

	typeStr := prov.TypeName()
	if typeStr != "dns" {
		t.Error("The string representation of provider type should be equal to 'dns'")
	}
}
