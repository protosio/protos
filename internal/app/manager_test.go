package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/mock"
	"github.com/protosio/protos/internal/util"

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
	cmMock := mock.NewMockCapabilityManager(ctrl)
	metaMock := mock.NewMockMeta(ctrl)
	pruMock := mock.NewMockPlatformRuntimeUnit(ctrl)
	appStoreMock := NewMockappStore(ctrl)
	capMock := mock.NewMockCapability(ctrl)

	c := make(chan interface{}, 10)

	// test app manager creation and initial app loading from db
	dbMock.EXPECT().GetMap(appDS, gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			apps := to.(*[]*App)
			*apps = append(*apps,
				&App{ID: "id1", Name: "app1", Status: statusUnknown, Tasks: []string{"1", "2"}, access: &sync.Mutex{}, PublicPorts: []util.Port{util.Port{Nr: 10000, Type: util.TCP}}},
				&App{ID: "id2", Name: "app2", Status: statusUnknown, Tasks: []string{"1"}, access: &sync.Mutex{}},
				&App{ID: "id3", Name: "app3", Status: statusUnknown, Tasks: []string{"1"}, access: &sync.Mutex{}})
		})

	// one of the inputs is nil
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A nil input in the CreateManager call should lead to a panic")
			}
		}()
		CreateManager(rmMock, nil, rpMock, dbMock, metaMock, wspMock, nil, cmMock)
	}()

	// happy case
	am := CreateManager(rmMock, tmMock, rpMock, dbMock, metaMock, wspMock, appStoreMock, cmMock)

	//
	// GetCopy
	//

	t.Run("GetCopy", func(t *testing.T) {
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
	})

	//
	// CopyAll
	//

	t.Run("CopyAll", func(t *testing.T) {
		if len(am.CopyAll()) != 3 {
			t.Errorf("CopyAll should return 3 apps. Instead it returned %d", len(am.CopyAll()))
		}
	})

	//
	// Read
	//

	t.Run("Read", func(t *testing.T) {
		_, err := am.Read("wrongId")
		if err == nil {
			t.Errorf("Read(wrongId) should return an error")
		}

		app, err := am.Read("id1")
		if err != nil {
			t.Errorf("Read(id1) should NOT return an error: %s", err.Error())
		} else {
			if app.GetName() != "app1" {
				t.Errorf("App id 'id1' should have name app1, NOT %s", app.GetName())
			}
		}
	})

	//
	// Select
	//
	t.Run("Select", func(t *testing.T) {
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
	})

	//
	// CreateAsync
	//

	t.Run("CreateAsync", func(t *testing.T) {
		// one of the required inputs is nil
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("An empty required input in the CreateAsync call should lead to a panic")
				}
			}()
			am.CreateAsync("", "0.0.1", "c", map[string]string{}, false)
		}()
		tmMock.EXPECT().New(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		_ = am.CreateAsync("a", "b", "c", map[string]string{}, false)
	})

	//
	// Create
	//

	t.Run("Create", func(t *testing.T) {
		_, err := am.Create("a", "b", "", map[string]string{}, core.InstallerMetadata{})
		if err == nil {
			t.Errorf("Creating an app using a blank name should result in an error")
		}

		// installer params test
		_, err = am.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{Params: []string{"test"}})
		if err == nil {
			t.Errorf("Creating an app and not providing the mandatory params should result in an error")
		}

		// capability test, error while creating DNS for app
		metaMock.EXPECT().GetPublicIP().Return("1.1.1.1").Times(1)
		rmMock.EXPECT().CreateDNS(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("test error")).Times(1)
		cmMock.EXPECT().GetByName("PublicDNS").Return(capMock, nil).Times(2)
		capMock.EXPECT().GetName().Return("PublicDNS").Times(1)
		cmMock.EXPECT().Validate(capMock, gomock.Any()).Return(true).Times(1)
		_, err = am.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{Capabilities: []string{"PublicDNS"}})
		if err == nil {
			t.Error("Creating an app and having a DNS creation error should result in an error")
		}

		// happy case
		wspMock.EXPECT().GetWSPublishChannel().Return(c).Times(1)
		tmMock.EXPECT().GetIDs(gomock.Any()).Return(*linkedhashmap.New()).Times(1)
		rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{}).Times(1)
		cmMock.EXPECT().GetByName("PublicDNS").Return(capMock, nil).Times(1)
		capMock.EXPECT().GetName().Return("PublicDNS").Times(1)
		dbMock.EXPECT().InsertInMap(appDS, gomock.Any(), gomock.Any()).Return(nil).Times(1)
		app, err := am.Create("a", "b", "c", map[string]string{}, core.InstallerMetadata{})

		dbMock.EXPECT().RemoveFromMap(appDS, gomock.Any()).Return(nil).Times(1)
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(1)
		rpMock.EXPECT().CleanUpSandbox(gomock.Any()).Return(nil).Times(1)
		pruMock.EXPECT().Remove().Return(nil).Times(1)
		am.Remove(app.GetID())
	})

	//
	// GetServices
	//
	t.Run("GetServices", func(t *testing.T) {
		dnsRscType := NewMockdnsResource(ctrl)
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
	})

	//
	// Remove
	//
	t.Run("Remove", func(t *testing.T) {
		initialNr := len(am.apps.apps)
		// non-existent app id
		err := am.Remove("id4")
		if err == nil {
			t.Error("Remove(id4) should return an error because id4 does not exist")
		}
		if len(am.apps.apps) != initialNr {
			t.Errorf("Wrong number of apps found: %d vs %d", len(am.apps.apps), initialNr)
		}

		// existent app id which returns an error on app.remove()
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(nil, fmt.Errorf("test error")).Times(1)
		err = am.Remove("id3")
		if err == nil {
			t.Error("Remove(id3) should return an error because app.remove() returns an error")
		}
		if len(am.apps.apps) != initialNr {
			t.Errorf("Wrong number of apps found: %d vs %d", len(am.apps.apps), initialNr)
		}

		// existent app id - happy path
		pruMock.EXPECT().Remove().Return(nil).Times(1)
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(1)
		rpMock.EXPECT().CleanUpSandbox(gomock.Any()).Return(nil).Times(1)
		dbMock.EXPECT().RemoveFromMap(appDS, gomock.Any()).Return(nil).Times(1)
		err = am.Remove("id2")
		if err != nil {
			t.Errorf("Remove(id2) should NOT return an error: %s", err.Error())
		}
		if len(am.apps.apps) != initialNr-1 {
			t.Errorf("Wrong number of apps found: %d vs %d", len(am.apps.apps), initialNr-1)
		}
	})

	//
	// RemoveAsync
	//
	t.Run("RemoveAsync", func(t *testing.T) {
		taskMock := mock.NewMockTask(ctrl)
		tmMock.EXPECT().New(gomock.Any(), gomock.Any()).Return(taskMock).Times(1)
		removeTask := am.RemoveAsync("id4")
		if removeTask != taskMock {
			t.Error("RemoveAsync returned an incorrect task")
		}
	})

	//
	// saveApp
	//
	t.Run("saveApp", func(t *testing.T) {
		app2 := &App{ID: "id2", Name: "app2", access: &sync.Mutex{}, parent: am}
		wspMock.EXPECT().GetWSPublishChannel().Return(c).Times(2)
		pruMock.EXPECT().GetStatus().Return("exited").Times(2)
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(2)
		tmMock.EXPECT().GetIDs(gomock.Any()).Return(*linkedhashmap.New()).Times(2)
		rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{}).Times(2)

		// happy path
		dbMock.EXPECT().InsertInMap(appDS, gomock.Any(), gomock.Any()).Return(nil).Times(1)
		am.saveApp(app2)

		// db error should lead to panic
		dbMock.EXPECT().InsertInMap(appDS, gomock.Any(), gomock.Any()).Return(errors.New("test db error")).Times(1)
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("A DB error in saveApp should lead to a panic")
				}
			}()
			am.saveApp(app2)
		}()
	})

	//
	// CreateDevApp
	//
	t.Run("CreateDevApp", func(t *testing.T) {
		// app creation returns error
		_, err := am.CreateDevApp("", core.InstallerMetadata{}, map[string]string{})
		if err == nil {
			t.Error("CreateDevApp should fail when the app creation step fails")
		}

		// happy case
		wspMock.EXPECT().GetWSPublishChannel().Return(c).Times(2)
		tmMock.EXPECT().GetIDs(gomock.Any()).Return(*linkedhashmap.New()).Times(2)
		rmMock.EXPECT().Select(gomock.Any()).Return(map[string]core.Resource{}).Times(2)
		cmMock.EXPECT().GetByName("PublicDNS").Return(capMock, nil).Times(1)
		capMock.EXPECT().GetName().Return("PublicDNS").Times(1)
		dbMock.EXPECT().InsertInMap(appDS, gomock.Any(), gomock.Any()).Return(nil).Times(2)
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(1)
		pruMock.EXPECT().GetStatus().Return("exited").Times(1)
		app, err := am.CreateDevApp("c", core.InstallerMetadata{}, map[string]string{})
		if err != nil {
			t.Errorf("CreateDevApp(...) should NOT return an error: %s", err.Error())
		}
		rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(1)
		rpMock.EXPECT().CleanUpSandbox(gomock.Any()).Return(nil).Times(1)
		pruMock.EXPECT().Remove().Return(nil).Times(1)
		dbMock.EXPECT().RemoveFromMap(appDS, gomock.Any()).Return(nil).Times(1)
		am.Remove(app.GetID())
	})

	//
	// GetAllPublic
	//

	// t.Run("GetAllPublic", func(t *testing.T) {
	// 	nrOfApps := len(am.apps.apps)
	// 	tasks := linkedhashmap.New()
	// 	tasks.Put("1", gomock.Any())
	// 	tasks.Put("2", gomock.Any())
	// 	tmMock.EXPECT().GetAll().Return(tasks).Times(1)
	// 	rpMock.EXPECT().GetSandbox(gomock.Any()).Return(pruMock, nil).Times(nrOfApps)
	// 	pruMock.EXPECT().GetStatus().Return("exited").Times(nrOfApps)
	// 	papps := am.GetAllPublic()
	// 	if len(papps) != nrOfApps {
	// 		t.Errorf("GetAllPublic() should return %d apps, but it returned %d", nrOfApps, len(papps))
	// 	}
	// 	tsks1 := linkedhashmap.Map(papps["id1"].(*PublicApp).Tasks)
	// 	if len(tsks1.Keys()) != 2 {
	// 		t.Errorf("There should be 2 tasks in the public app with id1, but there are %d", len(tsks1.Keys()))
	// 	}
	// 	tsks2 := linkedhashmap.Map(papps["id3"].(*PublicApp).Tasks)
	// 	if len(tsks2.Keys()) != 1 {
	// 		t.Errorf("There should be 1 tasks in the public app with id2, but there are %d", len(tsks2.Keys()))
	// 	}

	// })

}
