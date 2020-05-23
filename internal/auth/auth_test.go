package auth

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/golang/mock/gomock"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/mock"
	"github.com/protosio/protos/pkg/types"
)

func TestUser(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	cmMock := mock.NewMockCapabilityManager(ctrl)
	smMock := mock.NewMockSSHManager(ctrl)
	adminCapMock := mock.NewMockCapability(ctrl)
	um := CreateUserManager(dbMock, smMock, cmMock)

	user := &User{
		Username:     "testuser",
		Password:     "passwordhash",
		Name:         "First Last",
		IsDisabled:   false,
		Capabilities: []string{"UserAdmin"},
		parent:       um,
	}

	// GetUsername
	if user.GetUsername() != "testuser" {
		t.Errorf("GetUsername() returned '%s' instead of '%s'", user.GetUsername(), "testuser")
	}

	//
	// Save
	//

	// failed db save
	dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Return(errors.New("failed to save user to db")).Times(1)
	err := user.Save()
	if err == nil {
		t.Errorf("Save() should return an error")
	}

	// success
	dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	err = user.Save()
	if err != nil {
		t.Errorf("Save() should NOT return an error: %s", err.Error())
	}

	//
	// ValidateCapability
	//

	// fails to validate
	cmMock.EXPECT().Validate(adminCapMock, "UserAdmin").Return(false).Times(1)
	adminCapMock.EXPECT().GetName().Return("AdminCapMock").Times(1)
	err = user.ValidateCapability(adminCapMock)
	if err == nil {
		t.Error("ValidateCapability should return an error when passed a capability does not validate")
	}

	// success
	cmMock.EXPECT().Validate(adminCapMock, "UserAdmin").Return(true).Times(1)
	err = user.ValidateCapability(adminCapMock)
	if err != nil {
		t.Errorf("ValidateCapability should NOT return an error: %s", err.Error())
	}

	//
	// IsAdmin
	//

	// true
	cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
	cmMock.EXPECT().Validate(adminCapMock, "UserAdmin").Return(true).Times(1)
	if !user.IsAdmin() {
		t.Error("IsAdmin() should return true because the user has the UserAdmin capability")
	}

	// false
	user.Capabilities = []string{}
	cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
	adminCapMock.EXPECT().GetName().Return("AdminCapMock").Times(1)
	if user.IsAdmin() {
		t.Error("IsAdmin() should return false because the user does not have the UserAdmin capability")
	}

	//
	// GetInfo
	//
	cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(2)
	adminCapMock.EXPECT().GetName().Return("AdminCapMock").Times(2)
	userInfo := user.GetInfo()
	if userInfo.Username != user.Username || userInfo.IsAdmin != user.IsAdmin() || userInfo.Name != user.Name {
		t.Error("GetInfo return a struct with incorrect details")
	}

}

func TestUserManager(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	smMock := mock.NewMockSSHManager(ctrl)
	cmMock := mock.NewMockCapabilityManager(ctrl)

	// one of the inputs is nil
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A nil input in the CreateUserManager call should lead to a panic")
			}
		}()
		CreateUserManager(dbMock, nil, nil)
	}()

	um := CreateUserManager(dbMock, smMock, cmMock)

	//
	// CreateUser
	//

	t.Run("CreateUser", func(t *testing.T) {
		adminCapMock := mock.NewMockCapability(ctrl)
		devices := []types.UserDevice{}
		// short password
		_, err := um.CreateUser("username", "pass", "first last", "domain", false, devices)
		if err == nil {
			t.Error("CreateUser() should return an error when the password is shorter than 10 characters")
		}

		// successful non-admin
		dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Times(1)
		user, err := um.CreateUser("username", "longpassword", "first last", "domain", false, devices)
		if err != nil {
			t.Errorf("CreateUser() should NOT return an error: %s", err.Error())
		}
		cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
		adminCapMock.EXPECT().GetName().Return("UserAdmin").Times(1)
		if user.IsAdmin() == true {
			t.Error("CreateUser() should return non-admin user")
		}

		// successful admin
		dbMock.EXPECT().InsertInSet(gomock.Any(), gomock.Any()).Times(1)
		user, err = um.CreateUser("username", "longpassword", "first last", "domain", true, devices)
		if err != nil {
			t.Errorf("CreateUser() should NOT return an error: %s", err.Error())
		}
		cmMock.EXPECT().GetByName("UserAdmin").Return(adminCapMock, nil).Times(1)
		cmMock.EXPECT().Validate(adminCapMock, "UserAdmin").Return(true).Times(1)
		if user.IsAdmin() == false {
			t.Error("CreateUser() should return admin user")
		}
	})

	//
	// ValidateAndGetUser
	//

	t.Run("ValidateAndGetUser", func(t *testing.T) {
		// failed to retrieve user from db
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(errors.New("User db error")).Times(1)
		_, err := um.ValidateAndGetUser("user", "pass")
		if err == nil {
			t.Error("ValidateAndGetUser() should return an errror when the DB fails to retrieve the user")
		}

		// failed password comparison
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		_, err = um.ValidateAndGetUser("user", "pass")
		if err == nil {
			t.Error("ValidateAndGetUser() should return an errror when the passwords are different")
		}

		// user is disabled
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(table string, username string, to interface{}) {
			user := to.(*User)
			user.Username = username
			user.Password = "$2a$10$nV4sGvDTq0unZjTEjViGhO0/3wfl6FT32Nh1YLJbTtWQVxrnXF76i"
			user.IsDisabled = true
		})
		_, err = um.ValidateAndGetUser("user", "longpassword")
		if err == nil {
			t.Error("ValidateAndGetUser() should return an errror when the user is disabled")
		}

		// succesful user validation
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(table string, username string, to interface{}) {
			user := to.(*User)
			user.Username = username
			user.Password = "$2a$10$nV4sGvDTq0unZjTEjViGhO0/3wfl6FT32Nh1YLJbTtWQVxrnXF76i"
			user.IsDisabled = false
		})
		_, err = um.ValidateAndGetUser("user", "longpassword")
		if err != nil {
			t.Errorf("ValidateAndGetUser() should NOT return an errror: %s", err.Error())
		}

	})

	//
	// GetUser
	//

	t.Run("GetUser", func(t *testing.T) {
		// failed to retrieve user from db
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(errors.New("User db error")).Times(1)
		_, err := um.GetUser("user")
		if err == nil {
			t.Error("GetUser() should return an errror when the DB fails to retrieve the user")
		}

		// success
		dbMock.EXPECT().GetSet(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(table string, username string, to interface{}) {
			user := to.(*User)
			user.Username = username
			user.IsDisabled = false
		})
		_, err = um.GetUser("user")
		if err != nil {
			t.Errorf("GetUser() should NOT return an errror: %s", err.Error())
		}

	})

	//
	// SetParent
	//

	t.Run("SetParent", func(t *testing.T) {
		var userInterface core.User
		_, err := um.SetParent(userInterface)
		if err == nil {
			t.Error("SetParent() should return an error when the provided interface does not assert to a User struct")
		}

		userStruct := &User{}
		user, err := um.SetParent(userStruct)
		if err != nil {
			t.Errorf("SetParent() should NOT return an error: %s", err.Error())
		}
		usr, _ := user.(*User)
		if usr.parent == nil {
			t.Error("SetParent() should return an error with the parent variable set")
		}

	})

}
