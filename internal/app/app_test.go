package app

import (
	"fmt"
	"sync"
	"testing"

	"protos/internal/capability"
	"protos/internal/core"
	"protos/internal/mock"
	"protos/internal/util"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/golang/mock/gomock"
)

func TestAppManager(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rmMock := mock.NewMockResourceManager(ctrl)
	tmMock := mock.NewMockTaskManager(ctrl)
	rpMock := mock.NewMockRuntimePlatform(ctrl)
	dbMock := mock.NewMockDB(ctrl)
	wspMock := mock.NewMockWSPublisher(ctrl)
	metaMock := mock.NewMockMeta(ctrl)

	// test app manager creation and initial app loading from db
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			apps := to.(*[]*App)
			*apps = append(*apps,
				&App{ID: "id1", Name: "app1", access: &sync.Mutex{}, PublicPorts: []util.Port{util.Port{Nr: 10000, Type: util.TCP}}},
				&App{ID: "id2", Name: "app2", access: &sync.Mutex{}},
				&App{ID: "id3", Name: "app3", access: &sync.Mutex{}})
		})
	am := CreateManager(rmMock, tmMock, rpMock, dbMock, metaMock, wspMock)

	//
	// GetCopy
	//
	_, err := am.GetCopy("wrongId")
	if err == nil {
		t.Errorf("GetCopy(wrongId) should return an error")
	}

	app, err := am.GetCopy("id1")
	if err != nil {
		t.Errorf("GetCopy(id1) should NOT return an error: %s", err.Error())
	} else {
		if app.GetName() != "app1" {
			t.Errorf("App id 'id1' should have name app1, NOT %s", app.GetName())
		}
	}

	//
	// CopyAll
	//
	if len(am.CopyAll()) != 3 {
		t.Errorf("CopyAll should return 3 apps. Instead it returned %d", len(am.CopyAll()))
	}

	//
	// Read
	//
	_, err = am.Read("wrongId")
	if err == nil {
		t.Errorf("Read(wrongId) should return an error")
	}

	app, err = am.Read("id1")
	if err != nil {
		t.Errorf("Read(id1) should NOT return an error: %s", err.Error())
	} else {
		if app.GetName() != "app1" {
			t.Errorf("App id 'id1' should have name app1, NOT %s", app.GetName())
		}
	}

	//
	// Select
	//
	filter := func(app core.App) bool {
		if app.GetName() == "app2" {
			return true
		}
		return false
	}

	apps := am.Select(filter)
	if len(apps) != 1 {
		t.Errorf("Select(filter) should return 1 app. Instead it returned %d", len(apps))
	}
	for _, app := range apps {
		if app.GetName() != "app2" || app.GetID() != "id2" {
			t.Errorf("Expected app id '%s' and app name '%s', but found '%s' and '%s'", "id2", "app2", app.GetID(), app.GetName())
		}
	}

	//
	// CreateAsync
	//

	tmMock.EXPECT().New(gomock.Any()).Return(nil).Times(1)
	_ = am.CreateAsync("a", "b", "c", core.InstallerMetadata{}, map[string]string{}, false)

	//
	// Create
	//

	appMgr := am.(*Manager)
	_, err = appMgr.Create("a", "b", "", map[string]string{}, core.InstallerMetadata{}, "taskid")
	if err == nil {
		t.Errorf("Creating an app using a blank name should result in an error")
	}

	// installer params test
	_, err = appMgr.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{Params: []string{"test"}}, "taskid")
	if err == nil {
		t.Errorf("Creating an app and not providing the mandatory params should result in an error")
	}

	// capability test, error while creating DNS for app
	metaMock.EXPECT().GetPublicIP().Return("1.1.1.1").Times(1)
	rmMock.EXPECT().CreateDNS(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("test error")).Times(1)
	_, err = appMgr.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{Capabilities: []*capability.Capability{capability.PublicDNS}}, "taskid")
	if err == nil {
		t.Error("Creating an app and having a DNS creation error should result in an error")
	}

	// happy case
	c := make(chan interface{}, 10)
	wspMock.EXPECT().GetPublishChannel().Return(c).Times(1)
	tmMock.EXPECT().GetIDs(gomock.Any()).Return(linkedhashmap.Map{}).Times(1)
	rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{}).Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	_, err = appMgr.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{}, "taskid")

	//
	// GetServices
	//

	dnsRscType := mock.NewMockType(ctrl)
	dnsRscType.EXPECT().GetName().Return("app1").Times(2)
	dnsRscType.EXPECT().GetValue().Return("1.1.1.1").Times(1)
	rscMock := mock.NewMockResource(ctrl)
	rscMock.EXPECT().GetAppID().Return("id1").Times(1)
	rscMock.EXPECT().GetValue().Return(dnsRscType).Times(1)
	rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{"1": rscMock}).Times(1)
	metaMock.EXPECT().GetDomain().Return("giurgiu.io").Times(1)

	services := am.GetServices()
	if len(services) != 1 {
		t.Fatalf("GetServices should only return 1 service in this test, but %d were returned", len(services))
	}
	svc := services[0]
	if len(svc.Ports) != 1 ||
		svc.Ports[0].Nr != 10000 ||
		svc.Ports[0].Type != util.TCP ||
		svc.Name != "app1" ||
		svc.Domain != "app1.giurgiu.io" {
		t.Error("GetServices returned a service with incorrect values")
	}

	//
	// Remove
	//

	// non-existent app id
	err = am.Remove("id4")
	if err == nil {
		t.Error("Remove(id4) should return an error because id4 does not exist")
	}

	// existent app id which returns an error on app.remove()
	rpMock.EXPECT().GetDockerContainer(gomock.Any()).Return(nil, fmt.Errorf("test error")).Times(1)
	err = am.Remove("id3")
	if err == nil {
		t.Error("Remove(id3) should return an error because app.remove() returns an error")
	}

	// existent app id - happy path
	pruMock := mock.NewMockPlatformRuntimeUnit(ctrl)
	pruMock.EXPECT().Remove().Return(nil).Times(1)
	rpMock.EXPECT().GetDockerContainer(gomock.Any()).Return(pruMock, nil).Times(1)
	err = am.Remove("id2")
	if err != nil {
		t.Errorf("Remove(id2) should NOT return an error: %s", err.Error())
	}

	//
	// RemoveAsync
	//

	taskMock := mock.NewMockTask(ctrl)
	tmMock.EXPECT().New(gomock.Any()).Return(taskMock).Times(1)
	removeTask := am.RemoveAsync("id4")
	if removeTask != taskMock {
		t.Error("RemoveAsync returned an incorrect task")
	}

}
