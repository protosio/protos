package resource

import (
	"testing"

	"github.com/golang/mock/gomock"
	"protos/internal/core"
	"protos/internal/mock"
)

func TestResourceManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	dbMock.EXPECT().Register(gomock.Any()).Return().Times(3)
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			rs := to.(*[]Resource)
			*rs = append(*rs, Resource{ID: "0001"}, Resource{ID: "0002"}, Resource{ID: "0003"})
		})
	dbMock.EXPECT().Remove(gomock.Any()).Return(nil).Times(1)

	rm := CreateManager(dbMock)

	// test if GetAll returns the right number of elements
	if len(rm.GetAll(false)) != 3 {
		t.Error("rm.GetAll should return 3 elements, but it returned", len(rm.GetAll(false)))
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
	dbMock.EXPECT().Register(gomock.Any()).Return().Times(3)
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(5)

	rm := CreateManager(dbMock)
	rc := rm.(core.ResourceCreator)

	// test DNS resource creation
	rsc, err := rc.CreateDNS("appid1", "app1", "MX", "1.2.3.4", 300)
	if err != nil {
		t.Error("rm/rc.CreateDNS should not return an error:", err.Error())
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
	if rsc.GetType() != DNS {
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

	// test Certificate resource creation
	rsc, err = rc.CreateCert("appid1", []string{"protos.io"})
	if err != nil {
		t.Error("rm/rc.CreateCert should not return an error:", err.Error())
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
