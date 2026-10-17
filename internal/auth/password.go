package auth

import "golang.org/x/crypto/bcrypt"

// DummyHash is a valid bcrypt hash used when the username does not exist,
// so the failure path still pays a compare and does not reveal which field was wrong.
const DummyHash = "$2a$10$n/jQjuqKcDhVR5mMcsabQe7S3gfjSVRRI8uWHeD6Y7Eoa25Io5zti"

func CheckPassword(hash, password string) bool {
	if hash == "" {
		hash = DummyHash
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
