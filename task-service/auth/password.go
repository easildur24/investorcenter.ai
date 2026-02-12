package auth

import (
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = getBcryptCost()

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getBcryptCost() int {
	costStr := os.Getenv("BCRYPT_COST")
	if costStr == "" {
		return 12
	}
	cost, err := strconv.Atoi(costStr)
	if err != nil || cost < 10 || cost > 14 {
		return 12
	}
	return cost
}
