package application

import "golang.org/x/crypto/bcrypt"

func bcryptGenerateFromPassword(password []byte, cost int) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, cost)
}

func bcryptCompare(hash, password []byte) error {
	return bcrypt.CompareHashAndPassword(hash, password)
}
