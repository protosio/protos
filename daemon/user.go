package daemon

import (
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"

	"golang.org/x/crypto/bcrypt"
)

const (
	userBucket = "user"
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

// ValidateAndGetUser takes a username and password and returns the full User struct if credentials are valid
func ValidateAndGetUser(username string, password string) (User, error) {
	log.Debugf("Searching for username %s", username)
	user := User{}
	err := Gconfig.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(userBucket))
		v := b.Get([]byte(username))
		errInvalid := errors.New("Invalid credentials")
		if v == nil {
			log.Debugf("Can't find user %s", username)
			return errInvalid
		}

		err := json.Unmarshal(v, &user)
		if err != nil {
			log.Error(err)
			return errInvalid
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			log.Debugf("Invalid password for user %s", username)
			return errInvalid
		}

		if user.IsDisabled {
			log.Debugf("User %s is disabled", username)
			return errInvalid
		}

		log.Debugf("User %s logged in successfuly", username)
		return nil
	})
	if err != nil {
		return User{}, err
	}
	return user, nil
}
