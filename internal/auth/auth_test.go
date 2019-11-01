package auth

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/protosio/protos/internal/mock"
)

func TestUserManager(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	cmMock := mock.NewMockCapabilityManager(ctrl)

	// one of the inputs is nil
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A nil input in the CreateUserManager call should lead to a panic")
			}
		}()
		CreateUserManager(dbMock, nil)
	}()

	um := CreateUserManager(dbMock, cmMock)

	//
	// CreateUser
	//

	t.Run("CreateUser", func(t *testing.T) {
		adminCapMock := mock.NewMockCapability(ctrl)
		// short password
		_, err := um.CreateUser("username", "pass", "first last", false)
		if err == nil {
			t.Error("CreateUser() should return an error when the password is shorter than 10 characters")
		}

		// successful non-admin
		dbMock.EXPECT().Save(gomock.Any()).Times(1)
		user, err := um.CreateUser("username", "longpassword", "first last", false)
		if err != nil {
			t.Errorf("CreateUser() should NOT return an error: %s", err.Error())
		}
		cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
		adminCapMock.EXPECT().GetName().Return("UserAdmin").Times(1)
		if user.IsAdmin() == true {
			t.Error("CreateUser() should return non-admin user")
		}

		// successful admin
		dbMock.EXPECT().Save(gomock.Any()).Times(1)
		user, err = um.CreateUser("username", "longpassword", "first last", true)
		if err != nil {
			t.Errorf("CreateUser() should NOT return an error: %s", err.Error())
		}
		cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
		cmMock.EXPECT().Validate(adminCapMock, "UserAdmin").Return(true).Times(1)
		if user.IsAdmin() == false {
			t.Error("CreateUser() should return admin user")
		}
	})

}
