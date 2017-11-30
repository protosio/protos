package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"protos/database"
	"protos/util"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	userBucket = "user"
)

var log = util.Log

// User represents a Protos user
type User struct {
	Username   string `json:"username" storm:"id"`
	Password   string `json:"password"`
	IsAdmin    bool   `json:"isadmin"`
	IsDisabled bool   `json:"isdisabled"`
}

// readCredentials reads a username and password interactively
func readCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

// generatePasswordHash takes a string representing the raw password, and generates a hash
func generatePasswordHash(password string) (string, error) {

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

	passwordHash, err := generatePasswordHash(password)
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
	return database.Save(user)
}

// ValidateAndGetUser takes a username and password and returns the full User struct if credentials are valid
func ValidateAndGetUser(username string, password string) (User, error) {
	log.Debugf("Searching for username %s", username)

	errInvalid := errors.New("Invalid credentials")
	var user User
	err := database.One("Username", username, &user)
	if err != nil {
		log.Debugf("Can't find user %s (%s)", username, err)
		return User{}, errInvalid
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Debugf("Invalid password for user %s", username)
		return User{}, errInvalid
	}

	if user.IsDisabled {
		log.Debugf("User %s is disabled", username)
		return User{}, errInvalid
	}

	log.Debugf("User %s logged in successfuly", username)
	return user, nil
}

// InitAdmin creates and initial admin user
func InitAdmin() {
	username, clearpassword := readCredentials()
	user, err := CreateUser(username, clearpassword, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("User %s has been created.", user.Username)
}
