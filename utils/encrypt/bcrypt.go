package encrypt

import "golang.org/x/crypto/bcrypt"

func BcryptText(text string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(text), 10)
	return string(bytes), err
}

func CompareBcryptText(hash, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}