package auth

import (
	"golang.org/x/crypto/bcrypt"
	"os"
	"strconv"
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

// Helper to get bcrypt cost from env
func getBcryptCost() int {
	costStr := os.Getenv("BCRYPT_COST")
	if costStr == "" {
		return 12 // Default cost
	}
	cost, err := strconv.Atoi(costStr)
	if err != nil || cost < 10 || cost > 14 {
		return 12
	}
	return cost
}
