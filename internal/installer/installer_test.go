package installer

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/mock"
	"github.com/protosio/protos/internal/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

func TestParserFunctions(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cm := mock.NewMockCapabilityManager(ctrl)
	capabilityMock := mock.NewMockCapability(ctrl)

	cm.EXPECT().GetByName("ResourceProvider").Return(capabilityMock, nil).Times(1)
	cm.EXPECT().GetByName("WrongCap").Return(nil, errors.New("wrong capability")).Times(1)
	caps := validateInstallerCapabilities(cm, "ResourceProvider,WrongCap")
	if len(caps) != 1 {
		t.Errorf("Wrong number of capabilities returned. %d instead of 1", len(caps))
	}
	if caps[0] != "ResourceProvider" {
		t.Errorf("Wrong capability returned by the parse function")
	}

	ports := parsePublicPorts("1/TCP,2/UDP,sfdsf,80000/TCP,50/SH")
	if len(ports) != 2 {
		t.Errorf("Wrong number of ports returned. %d instead of 2", len(caps))
	}
	if ports[0].Nr != 1 || ports[0].Type != util.TCP || ports[1].Nr != 2 || ports[1].Type != util.UDP {
		t.Errorf("Wrong data in the ports array returned by the parsePublicPorts: %v", ports)
	}

}

func TestMetadata(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cm := mock.NewMockCapabilityManager(ctrl)
	capabilityMock := mock.NewMockCapability(ctrl)

	testMetadata := map[string]string{
		"protos.installer.metadata.capabilities": "ResourceProvider,ResourceConsumer,InternetAccess,GetInformation,PublicDNS,AuthUser",
		"protos.installer.metadata.requires":     "dns",
		"protos.installer.metadata.provides":     "mail,backup",
		"protos.installer.metadata.publicports":  "80/tcp,443/tcp,9999/udp",
		"protos.installer.metadata.name":         "testapp",
	}

	cm.EXPECT().GetByName(gomock.Any()).Return(capabilityMock, nil).Times(5)
	cm.EXPECT().GetByName(gomock.Any()).Return(nil, errors.New("wrong capability")).Times(1)
	_, err := parseMetadata(cm, testMetadata)
	if err == nil {
		t.Errorf("parseMetadata(testMetadata) should return an error on missing description")
	}

	testMetadata["protos.installer.metadata.description"] = "Small app description"

	cm.EXPECT().GetByName(gomock.Any()).Return(capabilityMock, nil).Times(5)
	cm.EXPECT().GetByName(gomock.Any()).Return(nil, errors.New("wrong capability")).Times(1)
	metadata, err := parseMetadata(cm, testMetadata)
	if err != nil {
		t.Errorf("parseMetadata(testMetadata) should not return an error, but it did: %s", err)
	}

	if len(metadata.PublicPorts) != 3 {
		t.Errorf("There should be %d publicports in the metadata. There are %d", 3, len(metadata.PublicPorts))
	}

	if (len(metadata.Requires) == 1 && metadata.Requires[0] != "dns") || len(metadata.Requires) != 1 {
		t.Errorf("metadata.Requires should only have 'dns' stored: %v", metadata.Requires)
	}

	if (len(metadata.Provides) == 2 && metadata.Provides[0] != "mail" && metadata.Provides[1] != "backup") || len(metadata.Provides) != 2 {
		t.Errorf("metadata.Provides should only have 'mail,backup' stored: %v", metadata.Requires)
	}

	if len(metadata.Capabilities) != 5 {
		t.Errorf("metadata.Capabilities should have 5 elements, but it has %d", len(metadata.Capabilities))
	}

}

func TestInstaller(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	installerParent := NewMockinstallerParent(ctrl)
	rpMock := mock.NewMockRuntimePlatform(ctrl)
	tmMock := mock.NewMockTaskManager(ctrl)

	inst := Installer{Name: "TestInstaller", ID: "id1", Versions: map[string]core.InstallerMetadata{"1.0": core.InstallerMetadata{PlatformID: "id1"}}, parent: installerParent}

	//
	// GetMetadata
	//

	t.Run("GetMetadata", func(t *testing.T) {
		// metadata for the supplied version does not exist
		_, err := inst.GetMetadata("2.0")
		if err == nil {
			t.Error("GetMetadata() should return an error when metadata for the supplied version does not exist")
		}

		// happy case
		_, err = inst.GetMetadata("1.0")
		if err != nil {
			t.Errorf("GetMetadata() should NOT return an error: %s", err.Error())
		}
	})

	//
	// Download
	//

	t.Run("Download", func(t *testing.T) {
		// metadata for the supplied version does not exist
		dt := DownloadTask{Version: "2.0"}
		err := inst.Download(dt)
		if err == nil {
			t.Error("Download() should return an error when metadata for the supplied version does not exist")
		}

		dt.Version = "1.0"

		// docker image download fails
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().PullDockerImage(dt.b, inst.Versions[dt.Version].PlatformID, inst.Name, dt.Version).Return(errors.New("error downloading docker image")).Times(1)
		err = inst.Download(dt)
		if err == nil {
			t.Error("Download() should return an error when the docker image download fails")
		}

		// happy case
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().PullDockerImage(dt.b, inst.Versions[dt.Version].PlatformID, inst.Name, dt.Version).Return(nil).Times(1)
		err = inst.Download(dt)
		if err != nil {
			t.Errorf("Download() should NOT return an error: %s", err.Error())
		}
	})

	//
	// DownloadAsync
	//

	t.Run("DownloadAsync", func(t *testing.T) {
		tskMock := mock.NewMockTask(ctrl)
		installerParent.EXPECT().getTaskManager().Return(tmMock).Times(1)
		tmMock.EXPECT().New("Download application installer", gomock.Any()).Return(tskMock).Times(1)
		task := inst.DownloadAsync("1.0", "id1")
		if task != tskMock {
			t.Errorf("DownloadAsync() returned the wrong task: %p vs %p", tskMock, task)
		}
	})

	//
	// IsPlatformImageAvailable
	//

	t.Run("IsPlatformImageAvailable", func(t *testing.T) {
		// metadata for the supplied version does not exist
		_, err := inst.IsPlatformImageAvailable("2.0")
		if err == nil {
			t.Error("IsPlatformImageAvailable() should return and error when the metadata is not available for an image version")
		}

		// error retrieving Docker image
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().GetDockerImage(inst.Versions["1.0"].PlatformID).Return(types.ImageInspect{}, errors.New("failed to retrieve image")).Times(1)
		_, err = inst.IsPlatformImageAvailable("1.0")
		if err == nil {
			t.Error("IsPlatformImageAvailable() should return an error when retrieving the image fails ")
		}

		// happy case
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().GetDockerImage(inst.Versions["1.0"].PlatformID).Return(types.ImageInspect{}, nil).Times(1)
		found, err := inst.IsPlatformImageAvailable("1.0")
		if err != nil {
			t.Errorf("IsPlatformImageAvailable() should not return an error: %s", err.Error())
		}
		if found == false {
			t.Error("IsPlatformImageAvailable() should return true")
		}
	})

	//
	// Remove
	//

	t.Run("Remove", func(t *testing.T) {
		// error removing Docker image
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().RemoveDockerImage(inst.Versions["1.0"].PlatformID).Return(errors.New("failed to remove image")).Times(1)
		err := inst.Remove()
		log.Info(err)
		if err == nil {
			t.Error("Remove() should return an error when removing the image fails")
		}

		// happy case
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().RemoveDockerImage(inst.Versions["1.0"].PlatformID).Return(nil).Times(1)
		err = inst.Remove()
		if err != nil {
			t.Errorf("Remove() should NOT return an error: %s", err.Error())
		}
	})
}

func TestInstallerCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rpMock := mock.NewMockRuntimePlatform(ctrl)
	appStore := &AppStore{rp: rpMock}

	// this will not compile if AppStore does not implement core.InstallerCache interface
	_ = core.InstallerCache(appStore)

	//
	// GetLocalInstallers
	//

	t.Run("GetLocalInstallers", func(t *testing.T) {
		// failed to retrieve Docker images
		rpMock.EXPECT().GetAllDockerImages().Return(nil, errors.New("failed to retrieve images"))
		_, err := appStore.GetLocalInstallers()
		if err == nil {
			t.Error("GetLocalInstallers() should return an error when retrieving all docker images fails")
		}

		// happy case
		images := map[string]types.ImageSummary{"id1": types.ImageSummary{RepoTags: []string{"imagename:v0.1"}}}
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		installers, err := appStore.GetLocalInstallers()
		if err != nil {
			t.Errorf("GetLocalInstallers() should NOT return an error: %s", err.Error())
		}
		if len(installers) != 1 {
			t.Errorf("GetLocalInstallers() returned an incorrect number of installers: %d instead of %d", len(installers), 1)
		}
		installerID := util.String2SHA1("imagename")
		if installers[installerID].(Installer).ID != installerID {
			t.Errorf("GetLocalInstallers() returned an installer with incorrect ID: %s instead of %s", installers["id1"].(Installer).ID, installerID)
		}
	})

	//
	// GetLocalInstaller
	//

	t.Run("GetLocalInstaller", func(t *testing.T) {
		installerID := util.String2SHA1("imagename")
		// failed to retrieve Docker images
		rpMock.EXPECT().GetAllDockerImages().Return(nil, errors.New("failed to retrieve images"))
		_, err := appStore.GetLocalInstaller("id1")
		if err == nil {
			t.Error("GetLocalInstaller() should return an error when retrieving all docker images fails")
		}

		images := map[string]types.ImageSummary{installerID: types.ImageSummary{ID: "imageID", RepoTags: []string{"imagename:v0.1"}}}

		// failed to retrieve Docker image (persitence path)
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		rpMock.EXPECT().GetDockerImage("imageID").Return(types.ImageInspect{}, errors.New("failed to retrieve image"))
		_, err = appStore.GetLocalInstaller(installerID)
		if err == nil {
			t.Error("GetLocalInstaller() should return an error when retrieving a specific docker images fails")
		}

		// failed to retrieve Docker image volume path
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		rpMock.EXPECT().GetDockerImage("imageID").Return(types.ImageInspect{}, nil)
		rpMock.EXPECT().GetDockerImageDataPath(gomock.Any()).Return("", errors.New("failed to retrieve image volume path"))
		_, err = appStore.GetLocalInstaller(installerID)
		if err == nil {
			t.Error("GetLocalInstaller() should return an error when retrieving the image volume path fails")
		}

		// happy path
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		rpMock.EXPECT().GetDockerImage("imageID").Return(types.ImageInspect{Config: &container.Config{Labels: map[string]string{"foo": "bar"}}}, nil)
		rpMock.EXPECT().GetDockerImageDataPath(gomock.Any()).Return("/data", nil)
		_, err = appStore.GetLocalInstaller(installerID)
		if err != nil {
			t.Errorf("GetLocalInstaller() should NOT return an error: %s", err.Error())
		}

	})

	//
	// RemoveLocalInstaller
	//

	t.Run("RemoveLocalInstaller", func(t *testing.T) {
		// failed to get images
		rpMock.EXPECT().GetAllDockerImages().Return(nil, errors.New("failed to retrieve images"))
		err := appStore.RemoveLocalInstaller("id1")
		if err == nil {
			t.Error("RemoveLocalInstaller() should return an error when it fails to retrieve the local installer")
		}

		installerID := util.String2SHA1("imagename")
		images := map[string]types.ImageSummary{installerID: types.ImageSummary{ID: "imageID", RepoTags: []string{"imagename:v0.1"}}}

		// failed to remove installer images
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		rpMock.EXPECT().GetDockerImage("imageID").Return(types.ImageInspect{ID: "imageID", Config: &container.Config{Labels: map[string]string{"foo": "bar"}}}, nil)
		rpMock.EXPECT().GetDockerImageDataPath(gomock.Any()).Return("/data", nil)
		rpMock.EXPECT().RemoveDockerImage("imageID").Return(errors.New("failed to delete image"))
		err = appStore.RemoveLocalInstaller(installerID)
		if err == nil {
			t.Error("RemoveLocalInstaller() should return an error when it fails to remove the local image")
		}

		// happy case
		rpMock.EXPECT().GetAllDockerImages().Return(images, nil)
		rpMock.EXPECT().GetDockerImage("imageID").Return(types.ImageInspect{ID: "imageID", Config: &container.Config{Labels: map[string]string{"foo": "bar"}}}, nil)
		rpMock.EXPECT().GetDockerImageDataPath(gomock.Any()).Return("/data", nil)
		rpMock.EXPECT().RemoveDockerImage("imageID").Return(nil)
		err = appStore.RemoveLocalInstaller(installerID)
		if err != nil {
			t.Errorf("RemoveLocalInstaller() should NOT return an error: %s", err.Error())
		}

	})

}

func TestAppStore(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rp := mock.NewMockRuntimePlatform(ctrl)
	tm := mock.NewMockTaskManager(ctrl)
	cm := mock.NewMockCapabilityManager(ctrl)
	clientMock := NewMockhttpClient(ctrl)
	capabilityMock := mock.NewMockCapability(ctrl)
	getHTTPClient = func() httpClient {
		return clientMock
	}

	// one of the inputs is nil
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A nil input in CreateAppStore call should lead to a panic")
			}
		}()
		CreateAppStore(nil, nil, nil)
	}()

	// happy case
	appStore := CreateAppStore(rp, tm, cm)
	if appStore.rp != rp {
		t.Errorf("appStore instance should have the same rp instance as the mock: %p vs %p", appStore.rp, rp)
	}

	//
	// GetInstallers
	//

	t.Run("GetInstallers", func(t *testing.T) {
		// http get request fails
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/all").Return(nil, errors.New("app store http request failure"))
		_, err := appStore.GetInstallers()
		if err == nil {
			t.Error("GetInstallers() should return an error when the http get request fails")
		}

		// http response is bad (not in the 200 range)
		body := ioutil.NopCloser(strings.NewReader("wrong http request"))
		resp := &http.Response{Status: "400 Bad Request", StatusCode: 400, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/all").Return(resp, nil)
		_, err = appStore.GetInstallers()
		if err == nil {
			t.Error("GetInstallers() should return an error when the http get request returns a bad response")
		}

		// json payload from http response is invalid (not an Installer)
		body = ioutil.NopCloser(strings.NewReader("{\"test\": \"value\"}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/all").Return(resp, nil)
		_, err = appStore.GetInstallers()
		if err == nil {
			t.Error("GetInstallers() should return an error when the http get request returns a bad response")
		}

		// happy case
		fd, err := os.Open("../mock/app_store_all_response.json")
		if err != nil {
			t.Fatal("Failed to open ../mock/app_store_all_response.json file")
		}
		body = ioutil.NopCloser(bufio.NewReader(fd))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		cm.EXPECT().GetByName(gomock.Any()).Return(capabilityMock, nil).Times(13)
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/all").Return(resp, nil)
		installers, err := appStore.GetInstallers()
		if err != nil {
			t.Fatalf("GetInstallers() should not return an error: %s", err.Error())
		}
		if len(installers) != 4 {
			t.Fatalf("GetInstallers() returned the wrong nr of installers: 4 vs %d", len(installers))
		}
		inst := installers["09eda098ec82bcf862df67933ef6451cdbab3a4b"].(Installer)
		if inst.Name != "mailu" {
			t.Errorf("GetInstallers() returned the wrong installer: %v", inst)
		}

	})

	//
	// GetInstaller
	//

	t.Run("GetInstaller", func(t *testing.T) {
		// http get request fails
		id := "id1"
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/"+id).Return(nil, errors.New("app store http request failure"))
		_, err := appStore.GetInstaller(id)
		if err == nil {
			t.Error("GetInstaller() should return an error when the http get request fails")
		}

		// http response is bad (not in the 200 range)
		body := ioutil.NopCloser(strings.NewReader("wrong http request"))
		resp := &http.Response{Status: "400 Bad Request", StatusCode: 400, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/"+id).Return(resp, nil)
		_, err = appStore.GetInstaller(id)
		if err == nil {
			t.Error("GetInstaller() should return an error when the http get request returns a bad response")
		}

		// json payload from http response is invalid
		body = ioutil.NopCloser(strings.NewReader("{\"test: \"value\"}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/"+id).Return(resp, nil)
		_, err = appStore.GetInstaller(id)
		if err == nil {
			t.Error("GetInstaller() should return an error when the http get request returns a bad response")
		}

		// happy case
		fd, err := os.Open("../mock/app_store_one_installer_response.json")
		if err != nil {
			t.Fatal("Failed to open ../mock/app_store_one_installer_response.json file")
		}
		body = ioutil.NopCloser(bufio.NewReader(fd))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/"+id).Return(resp, nil).Times(1)
		cm.EXPECT().GetByName(gomock.Any()).Return(capabilityMock, nil).Times(2)
		installer, err := appStore.GetInstaller(id)
		if err != nil {
			t.Errorf("GetInstaller() should not return an error: %s", err.Error())
		}
		inst := installer.(Installer)
		if inst.Name != "installer-name" {
			t.Errorf("GetInstaller() returned the wrong installer: %v", inst)
		}
		// test metadata decoding
		metadata, err := inst.GetMetadata("0.0.8")
		if err != nil {
			t.Errorf("installer.GetMetadata() should not return an error: %s", err.Error())
		}
		if len(metadata.Capabilities) != 2 {
			t.Errorf("installer.GetMetadata() returned metadata with the wrong number of capabilities: %d instead of 2", len(metadata.Capabilities))
		}
		if found, _ := util.StringInSlice("ResourceProvider", metadata.Capabilities); found != true {
			t.Error("installer.GetMetadata() should return metadata that contains capabilitity 'ResourceProvider'")
		}

	})

	//
	// Search
	//

	t.Run("Search", func(t *testing.T) {
		// http get request fails
		clientMock.EXPECT().Get("https://apps.protos.io/api/v1/search?key=value").Return(nil, errors.New("app store http request failure"))
		_, err := appStore.Search("key", "value")
		if err == nil {
			t.Error("Search() should return an error when the http get request fails")
		}

		// http response is bad (not in the 200 range)
		body := ioutil.NopCloser(strings.NewReader("wrong http request"))
		resp := &http.Response{Status: "400 Bad Request", StatusCode: 400, Body: body}
		clientMock.EXPECT().Get("https://apps.protos.io/api/v1/search?key=value").Return(resp, nil)
		_, err = appStore.Search("key", "value")
		if err == nil {
			t.Error("Search() should return an error when the http get request returns a bad response")
		}

		// json payload from http response is invalid (not an Installer)
		body = ioutil.NopCloser(strings.NewReader("{\"test\": \"value\"}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get("https://apps.protos.io/api/v1/search?key=value").Return(resp, nil)
		_, err = appStore.Search("key", "value")
		if err == nil {
			t.Error("Search() should return an error when the http get request returns a bad response")
		}

		// happy case
		fd, err := os.Open("../mock/app_store_search_response.json")
		if err != nil {
			t.Fatal("Failed to open ../mock/app_store_search_response.json file")
		}
		body = ioutil.NopCloser(bufio.NewReader(fd))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		cm.EXPECT().GetByName(gomock.Any()).Return(capabilityMock, nil).Times(2)
		clientMock.EXPECT().Get("https://apps.protos.io/api/v1/search?key=value").Return(resp, nil)
		installers, err := appStore.Search("key", "value")
		if err != nil {
			t.Fatalf("Search() should not return an error: %s", err.Error())
		}
		if len(installers) != 1 {
			t.Fatalf("Search() returned the wrong nr of installers: 1 vs %d", len(installers))
		}
		inst := installers["924bbbfeabb039828c0066ab90b2bfa8cde41024"].(Installer)
		if inst.Name != "namecheap-dns" {
			t.Errorf("Search() returned the wrong installer: %v", inst)
		}

	})

	//
	// CreateTemporaryInstaller
	//

	t.Run("CreateTemporaryInstaller", func(t *testing.T) {
		version := map[string]core.InstallerMetadata{}
		instInterface := appStore.CreateTemporaryInstaller("testName", version)
		inst := instInterface.(*Installer)
		if inst.parent == nil || inst.ID == "" {
			t.Errorf("CreateTemporaryInstaller() should return an Installer with all the required fields in place: '%v'", inst)
		}
	})

}

func TestTask(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	p := mock.NewMockProgress(ctrl)
	parent := NewMockinstallerParent(ctrl)
	rpMock := mock.NewMockRuntimePlatform(ctrl)
	taskMock := mock.NewMockTask(ctrl)
	inst := Installer{ID: "installer1", Name: "installer-name", parent: parent}
	task := DownloadTask{Inst: inst, AppID: "app1", Version: "0.1"}

	//
	// Run
	//

	// installer fails to download
	taskMock.EXPECT().AddApp("app1").Times(1)
	taskMock.EXPECT().Save().Times(1)
	err := task.Run(taskMock, "id1", p)
	if err == nil {
		t.Error("Run() should return an error when the installer fails to download")
	}

	// happy case
	task.Inst.Versions = map[string]core.InstallerMetadata{"0.1": core.InstallerMetadata{}}
	taskMock.EXPECT().AddApp("app1").Times(1)
	taskMock.EXPECT().Save().Times(1)
	parent.EXPECT().getPlatform().Return(rpMock).Times(1)
	rpMock.EXPECT().PullDockerImage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	err = task.Run(taskMock, "id1", p)
	if err != nil {
		t.Errorf("Run() should NOT return an error: %s", err.Error())
	}

}
