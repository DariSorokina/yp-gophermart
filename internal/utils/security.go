package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) string {
	passwordBytes := []byte(password)
	hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	if err != nil {
		log.Print(err.Error())
	}
	return string(hash)
}

func CheckPassword(hashedPassword, userPassword string) error {
	hashedPasswordBytes := []byte(hashedPassword)
	userPasswordBytes := []byte(userPassword)

	err := bcrypt.CompareHashAndPassword(hashedPasswordBytes, userPasswordBytes)
	return err

}
