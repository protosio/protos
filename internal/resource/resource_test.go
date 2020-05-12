package resource

import (
	"testing"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/mock"

	"github.com/golang/mock/gomock"
)

func TestResourceManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rscVal := &DNSResource{Host: "protos.io"}
	dbMock := mock.NewMockDB(ctrl)
	dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			rs := to.(*[]Resource)
			*rs = append(*rs, Resource{ID: "0001"}, Resource{ID: "0002"}, Resource{ID: "0003"})
		})
	dbMock.EXPECT().RemoveFromSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	// one of the inputs is nil
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A nil input in the CreateManager call should lead to a panic")
			}
		}()
		CreateManager(nil)
	}()

	rm := CreateManager(dbMock)

	//
	// Create
	//

	t.Run("Create", func(t *testing.T) {
		dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Times(1)
		_, err := rm.Create(core.DNS, rscVal, "testApp")
		if err != nil {
			t.Errorf("Create should NOT return an error: %s", err.Error())
		}

		_, err = rm.Create(core.DNS, rscVal, "secondApp")
		if err == nil {
			t.Error("Create should return an error when a resource with the same hash already exists")
		}
	})

	//
	// CreateFromJSON
	//

	t.Run("CreateFromJSON", func(t *testing.T) {
		jsonResource1 := []byte("{\"type\": \"dns\", \"value\": {\"host: \"protos.io\"}}")
		_, err := rm.CreateFromJSON(jsonResource1, "testApp")
		log.Info(err)
		if err == nil {
			t.Error("CreateFromJSON should return an error when a invalid JSON is passed")
		}

		jsonResource2 := []byte("{\"type\": \"dns\", \"value\": {\"host\": \"protos.io\"}}")
		_, err = rm.CreateFromJSON(jsonResource2, "testApp")
		if err == nil {
			t.Error("CreateFromJSON should return an error when a resource with the same hash already exists")
		}

		dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Times(1)
		jsonResource3 := []byte("{\"type\": \"dns\", \"value\": {\"host\": \"protos.com\"}}")
		_, err = rm.CreateFromJSON(jsonResource3, "testApp")
		if err != nil {
			t.Errorf("CreateFromJSON should NOT return an error: %s", err.Error())
		}

	})

	//
	// Select
	//

	t.Run("Select", func(t *testing.T) {
		selector := func(rsc core.Resource) bool {
			if rsc.GetID() == "0001" {
				return true
			}
			return false
		}
		resources := rm.Select(selector)
		if len(resources) != 1 {
			t.Errorf("There should only be 1 element in the selected resource list but there are %d", len(resources))
		}
	})

	// test if GetAll returns the right number of elements
	if len(rm.GetAll(false)) != 5 {
		t.Error("rm.GetAll should return 5 elements, but it returned", len(rm.GetAll(false)))
	}
	// if a non-existent resources is requested, an error should be returned
	_, err := rm.Get("bogus")
	if err == nil {
		t.Error("Get(bogus) should return a resource not found error")
	}
	// if an existent resources is requested, err should be nil, and rsc should have the correct id
	rsc, err := rm.Get("0002")
	if err != nil {
		t.Error("Get(1234) should return a valid resource")
	}
	if rsc.GetID() != "0002" {
		t.Error("Rsc should have id 0002 but it has", rsc.GetID())
	}
	// if a non-existent resource is deleted, err should NOT be nil
	err = rm.Delete("bogus")
	if err == nil {
		t.Error("Delete(bogus) should return an error")
	}
	// if an existing resource is deleted, err should be nil
	err = rm.Delete("0002")
	if err != nil {
		t.Error("Delete(0002) should NOT return an error")
	}
	// if a previously deleted resource is requested, err should NOT be nil
	_, err = rm.Get("0002")
	if err == nil {
		t.Error("Get(0002) should return an error because it was previously deleted")
	}

}

func TestResoureCreatorAndResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Return(nil).Times(5)

	rm := CreateManager(dbMock)

	//
	// CreateDNS
	//

	// test DNS resource creation
	rsc, err := rm.CreateDNS("appid1", "app1", "MX", "1.2.3.4", 300)
	if err != nil {
		t.Error("rm/rc.CreateDNS should not return an error:", err.Error())
	}

	// DNS resource creation should fail because of duplications
	_, err = rm.CreateDNS("appid1", "app1", "MX", "1.2.3.4", 300)
	if err == nil {
		t.Errorf("rm/rc.CreateDNS should return an error because of a duplicate DNS resource")
	}

	//
	// Resource related tests
	//

	rscstruct := rsc.(*Resource)
	// AppID should be equal to what was provided when the rsc was created
	if rsc.GetAppID() != "appid1" {
		t.Error("AppID should be appid1 but is", rsc.GetAppID())
	}
	// Resource type should be equal to what was provided when the rsc was created
	if rsc.GetType() != core.DNS {
		t.Error("Resource type should be dns but is", rsc.GetType())
	}
	// Resource should have status created
	if rscstruct.Status != core.Requested {
		t.Error("Resource status should be requested, but is", rscstruct.Status)
	}
	rsc.SetStatus(core.Created)
	if rscstruct.Status != core.Created {
		t.Error("Resource status should be created, but is", rscstruct.Status)
	}

	// Test DNS resource values
	rscval := rsc.GetValue().(*DNSResource)
	if rscval.Host != "app1" || rscval.Type != "MX" || rscval.Value != "1.2.3.4" || rscval.TTL != 300 {
		t.Error("DNS details should be (app1 MX 1.2.3.4 300) but are (", rscval.Host, rscval.Type, rscval.Value, rscval.TTL, ")")
	}

	// test resource UpdateValue
	rscval.Host = "app2"
	rsc.UpdateValue(rscval)
	rscval2 := rsc.GetValue().(*DNSResource)
	if rscval2.Host != "app2" {
		t.Error("rsc.UpdateValue failed to update value correctly. Host should be app2 but is", rscval2.Host)
	}

	//
	// CreateCert
	//

	// test Certificate resource creation
	rsc, err = rm.CreateCert("appid1", []string{"protos.io"})
	if err != nil {
		t.Error("rm/rc.CreateCert should not return an error:", err.Error())
	}

	// Certificate resource creation should fail because of duplications
	_, err = rm.CreateCert("appid1", []string{"protos.io"})
	if err == nil {
		t.Errorf("rm/rc.CreateCert should return an error because of a duplicate Certificate resource")
	}

	// test Certificate resource values
	cert := rsc.GetValue().(*CertificateResource)
	if cert.Domains[0] != "protos.io" {
		t.Error("Certificate details should be (protos.io) but are (", cert.Domains[0], ")")
	}
	// test resource sanitize
	cert.PrivateKey = []byte("secret")
	rsc.UpdateValue(cert)
	srsc := rsc.Sanitize()
	cert = srsc.GetValue().(*CertificateResource)
	if len(cert.PrivateKey) != 0 {
		t.Error("Certificate resource shuld sanitize the privatekey")
	}
}
