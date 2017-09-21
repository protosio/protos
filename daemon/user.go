package daemon

import (
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"

	"golang.org/x/crypto/bcrypt"
)

// User represents a Protos user
type User struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsAdmin    bool   `json:"isadmin"`
	IsDisabled bool   `json:"isdisabled"`
}

// GeneratePasswordHash takes a string representing the raw password, and generates a hash
func GeneratePasswordHash(password string) (string, error) {

	if len([]rune(password)) < 10 {
		return "", errors.New("Minimum password length is 10 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// CreateUser creates and returns a user
func CreateUser(username string, password string, isadmin bool) (User, error) {

	passwordHash, err := GeneratePasswordHash(password)
	if err != nil {
		return User{}, err
	}

	user := User{
		Username:   username,
		Password:   passwordHash,
		IsAdmin:    isadmin,
		IsDisabled: false,
	}

	return user, user.Save()
}

// Save saves the User struct to the database. The username is used as an unique key
func (user *User) Save() error {
	log.Debugf("Writing username %s to database", user.Username)
	return Gconfig.Db.Update(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("user"))

		userbuf, err := json.Marshal(user)
		if err != nil {
			return err
		}

		err = userBucket.Put([]byte(user.Username), userbuf)
		if err != nil {
			return err
		}

		return nil
	})
}
