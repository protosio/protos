package auth

import (
	"errors"
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

	//
	// ValidateAndGetUser
	//

	t.Run("ValidateAndGetUser", func(t *testing.T) {
		// failed to retrieve user from db
		dbMock.EXPECT().One(gomock.Any(), "user", gomock.Any()).Return(errors.New("User db error")).Times(1)
		_, err := um.ValidateAndGetUser("user", "pass")
		if err == nil {
			t.Error("ValidateAndGetUser() should return an errror when the DB fails to retrieve the user")
		}

		// failed password comparison
		dbMock.EXPECT().One(gomock.Any(), "user", gomock.Any()).Return(nil).Times(1)
		_, err = um.ValidateAndGetUser("user", "pass")
		if err == nil {
			t.Error("ValidateAndGetUser() should return an errror when the passwords are different")
		}

		// user is disabled
		dbMock.EXPECT().One(gomock.Any(), "user", gomock.Any()).Return(nil).Times(1).Do(func(table string, username string, to interface{}) {
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
		dbMock.EXPECT().One(gomock.Any(), "user", gomock.Any()).Return(nil).Times(1).Do(func(table string, username string, to interface{}) {
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
		dbMock.EXPECT().One(gomock.Any(), "user", gomock.Any()).Return(errors.New("User db error")).Times(1)
		_, err := um.GetUser("user")
		log.Info(err)
		if err == nil {
			t.Error("GetUser() should return an errror when the DB fails to retrieve the user")
		}

	})

}
