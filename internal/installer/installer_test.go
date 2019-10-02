package installer

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"protos/internal/capability"
	"protos/internal/core"
	"protos/internal/mock"
	"protos/internal/util"

	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

func TestParserFunctions(t *testing.T) {

	caps := parseInstallerCapabilities("ResourceProvider,WrongCap")
	if len(caps) != 1 {
		t.Errorf("Wrong number of capabilities returned. %d instead of 1", len(caps))
	}
	if caps[0] != capability.ResourceProvider {
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

	testMetadata := map[string]string{
		"protos.installer.metadata.capabilities": "ResourceProvider,ResourceConsumer,InternetAccess,GetInformation,PublicDNS,AuthUser",
		"protos.installer.metadata.requires":     "dns",
		"protos.installer.metadata.provides":     "mail,backup",
		"protos.installer.metadata.publicports":  "80/tcp,443/tcp,9999/udp",
		"protos.installer.metadata.name":         "testapp",
	}

	_, err := parseMetadata(testMetadata)
	if err == nil {
		t.Errorf("parseMetadata(testMetadata) should return an error on missing description")
	}

	testMetadata["protos.installer.metadata.description"] = "Small app description"

	metadata, err := parseMetadata(testMetadata)
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
		tmMock.EXPECT().New(gomock.Any()).Return(tskMock).Times(1)
		task := inst.DownloadAsync(tmMock, "1.0", "id1")
		if task != tskMock {
			t.Errorf("DownloadAsync() returned the wrong task: %p vs %p", tskMock, task)
		}
	})

	//
	// IsPlatformImageAvailable
	//

	t.Run("IsPlatformImageAvailable", func(t *testing.T) {
		// metadata for the supplied version does not exist
		if inst.IsPlatformImageAvailable("2.0") {
			t.Error("IsPlatformImageAvailable() should return false when the metadata is not available for an image version")
		}

		// error retrieving Docker image
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().GetDockerImage(inst.Versions["1.0"].PlatformID).Return(types.ImageInspect{}, errors.New("failed to retrieve image")).Times(1)
		if inst.IsPlatformImageAvailable("1.0") {
			t.Error("IsPlatformImageAvailable() should return false when retrieving the image fails ")
		}

		// happy case
		installerParent.EXPECT().getPlatform().Return(rpMock).Times(1)
		rpMock.EXPECT().GetDockerImage(inst.Versions["1.0"].PlatformID).Return(types.ImageInspect{}, nil).Times(1)
		if inst.IsPlatformImageAvailable("1.0") == false {
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

func TestAppStore(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rp := mock.NewMockRuntimePlatform(ctrl)
	clientMock := NewMockhttpClient(ctrl)
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
		CreateAppStore(nil)
	}()

	// happy case
	appStore := CreateAppStore(rp)
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
		body = ioutil.NopCloser(strings.NewReader("{\"id1\": {\"name\": \"installer name\", \"ID\": \"id1\"}}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/all").Return(resp, nil)
		installers, err := appStore.GetInstallers()
		if err != nil {
			t.Errorf("GetInstallers() should not return an error: %s", err.Error())
		}
		if len(installers) != 1 {
			t.Errorf("GetInstallers() returned the wrong nr of installers: 1 vs %d", len(installers))
		}
		inst := installers["id1"].(Installer)
		if inst.Name != "installer name" {
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
		body = ioutil.NopCloser(strings.NewReader("{\"name\": \"installer name\", \"ID\": \"id1\"}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get(gconfig.AppStoreURL+"/api/v1/installers/"+id).Return(resp, nil)
		installer, err := appStore.GetInstaller(id)
		if err != nil {
			t.Errorf("GetInstaller() should not return an error: %s", err.Error())
		}
		inst := installer.(Installer)
		if inst.Name != "installer name" {
			t.Errorf("GetInstaller() returned the wrong installer: %v", inst)
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
		body = ioutil.NopCloser(strings.NewReader("{\"id1\": {\"name\": \"installer name\", \"ID\": \"id1\"}}"))
		resp = &http.Response{Status: "200 OK", StatusCode: 200, Body: body}
		clientMock.EXPECT().Get("https://apps.protos.io/api/v1/search?key=value").Return(resp, nil)
		installers, err := appStore.Search("key", "value")
		if err != nil {
			t.Errorf("Search() should not return an error: %s", err.Error())
		}
		if len(installers) != 1 {
			t.Errorf("Search() returned the wrong nr of installers: 1 vs %d", len(installers))
		}
		inst := installers["id1"].(Installer)
		if inst.Name != "installer name" {
			t.Errorf("Search() returned the wrong installer: %v", inst)
		}

	})

}
