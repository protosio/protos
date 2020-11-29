package app

// import (
// 	"testing"

// 	"github.com/protosio/protos/internal/core"
// 	"github.com/protosio/protos/internal/mock"

// 	"github.com/golang/mock/gomock"
// 	"github.com/pkg/errors"
// )

// func TestTask(t *testing.T) {

// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	amMock := NewMocktaskParent(ctrl)

// 	t.Run("CreateAppTask", func(t *testing.T) {
// 		task := CreateAppTask{
// 			am:               nil,
// 			InstallerID:      "1",
// 			InstallerVersion: "",
// 			AppName:          "testapp",
// 			InstallerParams:  map[string]string{},
// 			StartOnCreation:  false,
// 		}
// 		tskID := "1"
// 		p := mock.NewMockProgress(ctrl)
// 		store := NewMockappStore(ctrl)
// 		inst := mock.NewMockInstaller(ctrl)
// 		app := NewMockapp(ctrl)
// 		baseTaskMock := mock.NewMockTask(ctrl)
// 		downloadTaskMock := mock.NewMockTask(ctrl)
// 		startAsyncTaskMock := mock.NewMockTask(ctrl)
// 		pruMock := mock.NewMockPlatformRuntimeUnit(ctrl)

// 		msgImageFound := "Container image found locally"
// 		msgCreated := "Created container"

// 		//
// 		// Run
// 		//

// 		// required inputs are missing for the task
// 		err := task.Run(baseTaskMock, tskID, p)
// 		log.Info(err)
// 		if err == nil {
// 			t.Error("Run() should return an error when one of the required task fields is empty")
// 		}
// 		task.am = amMock
// 		task.InstallerVersion = "0.0.0-dev"

// 		// failed to get installer metadata from the store
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(nil, errors.New("failed to retrieve image")).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the retrieval of the installer from the store fails")
// 		}

// 		// failed to retrieve installer metadata
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, errors.New("failed to retrieve install metadata")).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the retrieval of the installer metadata fails")
// 		}

// 		// app manager fails to create app
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(nil, errors.New("failed to create app")).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the app manager fails to create the app")
// 		}

// 		// image not available locally and download task returns an error
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(app, nil).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		p.EXPECT().SetPercentage(10).Times(1)
// 		p.EXPECT().SetState("Created application").Times(1)
// 		inst.EXPECT().IsPlatformImageAvailable(task.InstallerVersion).Return(false, nil).Times(1)
// 		app.EXPECT().GetID().Return("appid1").Times(1)
// 		inst.EXPECT().DownloadAsync(task.InstallerVersion, "appid1").Return(downloadTaskMock).Times(1)
// 		downloadTaskMock.EXPECT().GetID().Return("2")
// 		app.EXPECT().AddTask("2").Times(1)
// 		downloadTaskMock.EXPECT().Wait().Return(errors.New("download task error"))
// 		app.EXPECT().SetStatus(statusFailed).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the download image task fails")
// 		}

// 		// image available locally and create container fails
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(app, nil).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		p.EXPECT().SetPercentage(10).Times(1)
// 		p.EXPECT().SetState("Created application").Times(1)
// 		inst.EXPECT().IsPlatformImageAvailable(task.InstallerVersion).Return(true, nil).Times(1)
// 		p.EXPECT().SetPercentage(50).Times(1)
// 		p.EXPECT().SetState(msgImageFound).Times(1)
// 		app.EXPECT().createSandbox().Return(nil, errors.New("failed to create container"))
// 		app.EXPECT().SetStatus(statusFailed).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the app container fails to be created")
// 		}

// 		// start on creation is true and app fails to start
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(app, nil).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		p.EXPECT().SetPercentage(10).Times(1)
// 		p.EXPECT().SetState("Created application").Times(1)
// 		inst.EXPECT().IsPlatformImageAvailable(task.InstallerVersion).Return(true, nil).Times(1)
// 		p.EXPECT().SetPercentage(50).Times(1)
// 		p.EXPECT().SetState(msgImageFound).Times(1)
// 		app.EXPECT().createSandbox().Return(pruMock, nil)
// 		p.EXPECT().SetPercentage(70)
// 		p.EXPECT().SetState(msgCreated)
// 		task.StartOnCreation = true
// 		app.EXPECT().StartAsync().Return(startAsyncTaskMock).Times(1)
// 		startAsyncTaskMock.EXPECT().GetID().Return(tskID).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		startAsyncTaskMock.EXPECT().Wait().Return(errors.New("failed to start app")).Times(1)
// 		app.EXPECT().SetStatus(statusFailed).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err == nil {
// 			t.Error("Run() should return an error when the app fails to start")
// 		}

// 		// happy case, start on creation is false, installer metadata is nil, docker image is available locally
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(app, nil).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		p.EXPECT().SetPercentage(10).Times(1)
// 		p.EXPECT().SetState("Created application").Times(1)
// 		inst.EXPECT().IsPlatformImageAvailable(task.InstallerVersion).Return(true, nil).Times(1)
// 		p.EXPECT().SetPercentage(50).Times(1)
// 		p.EXPECT().SetState(msgImageFound).Times(1)
// 		app.EXPECT().createSandbox().Return(pruMock, nil)
// 		p.EXPECT().SetPercentage(70)
// 		p.EXPECT().SetState(msgCreated)
// 		task.StartOnCreation = false
// 		app.EXPECT().SetStatus(statusRunning).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err != nil {
// 			t.Errorf("Run() should NOT return an error: %s", err.Error())
// 		}

// 		// happy case, installer metadata is available, start on creation is true, docker image is available locally
// 		amMock.EXPECT().getAppStore().Return(store).Times(1)
// 		store.EXPECT().GetInstaller(task.InstallerID).Return(inst, nil).Times(1)
// 		inst.EXPECT().GetMetadata(task.InstallerVersion).Return(core.InstallerMetadata{}, nil).Times(1)
// 		amMock.EXPECT().createAppForTask(task.InstallerID, task.InstallerVersion, task.AppName, task.InstallerParams, core.InstallerMetadata{}, tskID).Return(app, nil).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		p.EXPECT().SetPercentage(10).Times(1)
// 		p.EXPECT().SetState("Created application").Times(1)
// 		// container image download
// 		inst.EXPECT().IsPlatformImageAvailable(task.InstallerVersion).Return(true, nil).Times(1)
// 		p.EXPECT().SetPercentage(50).Times(1)
// 		p.EXPECT().SetState(msgImageFound).Times(1)
// 		// create container
// 		app.EXPECT().createSandbox().Return(pruMock, nil)
// 		p.EXPECT().SetPercentage(70)
// 		p.EXPECT().SetState(msgCreated)
// 		// start on boot
// 		task.StartOnCreation = true
// 		app.EXPECT().StartAsync().Return(startAsyncTaskMock).Times(1)
// 		startAsyncTaskMock.EXPECT().GetID().Return(tskID).Times(1)
// 		app.EXPECT().AddTask(tskID).Times(1)
// 		startAsyncTaskMock.EXPECT().Wait().Return(nil).Times(1)
// 		// set status running
// 		app.EXPECT().SetStatus(statusRunning).Times(1)
// 		err = task.Run(baseTaskMock, tskID, p)
// 		if err != nil {
// 			t.Errorf("Run() should NOT return an error: %s", err.Error())
// 		}

// 	})

// 	t.Run("StartAppTask", func(t *testing.T) {

// 		p := mock.NewMockProgress(ctrl)
// 		baseTaskMock := mock.NewMockTask(ctrl)
// 		app := NewMockapp(ctrl)
// 		task := StartAppTask{
// 			app: app,
// 		}

// 		//
// 		// Run
// 		//

// 		p.EXPECT().SetPercentage(50).Times(1)
// 		app.EXPECT().AddTask("1").Times(1)
// 		app.EXPECT().Start().Times(1)
// 		err := task.Run(baseTaskMock, "1", p)
// 		if err != nil {
// 			t.Errorf("Run() should NOT return an error: %s", err.Error())
// 		}
// 	})

// 	t.Run("StopAppTask", func(t *testing.T) {

// 		p := mock.NewMockProgress(ctrl)
// 		baseTaskMock := mock.NewMockTask(ctrl)
// 		app := NewMockapp(ctrl)
// 		task := StopAppTask{
// 			app: app,
// 		}

// 		//
// 		// Run
// 		//

// 		p.EXPECT().SetPercentage(50).Times(1)
// 		app.EXPECT().AddTask("1").Times(1)
// 		app.EXPECT().Stop().Times(1)
// 		err := task.Run(baseTaskMock, "1", p)
// 		if err != nil {
// 			t.Errorf("Run() should NOT return an error: %s", err.Error())
// 		}
// 	})

// 	t.Run("RemoveAppTask", func(t *testing.T) {

// 		p := mock.NewMockProgress(ctrl)
// 		baseTaskMock := mock.NewMockTask(ctrl)
// 		task := RemoveAppTask{
// 			am:    nil,
// 			appID: "1",
// 		}

// 		//
// 		// Run
// 		//

// 		// application manager is nil
// 		func() {
// 			defer func() {
// 				r := recover()
// 				if r == nil {
// 					t.Errorf("RemoveAppTask should panic when the am field is not set")
// 				}
// 			}()
// 			task.Run(baseTaskMock, "1", p)
// 		}()

// 		task.am = amMock
// 		p.EXPECT().SetState("Deleting application").Times(1)
// 		p.EXPECT().SetPercentage(50).Times(1)
// 		amMock.EXPECT().Remove("1")
// 		err := task.Run(baseTaskMock, "1", p)
// 		if err != nil {
// 			t.Errorf("Run() should NOT return an error: %s", err.Error())
// 		}
// 	})

// }
