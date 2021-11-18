package cipherText

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// create hash with MD5 algorithm
func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// create hash with SHA256 algorithm
func createSHA256Hash(key string) []byte {
	hasher := sha256.Sum256([]byte(key))
	return hasher[:]
}
func EncryptData(data []byte, passphrase string) ([]byte, error) {
	// key := createHash(passphrase)
	key := createSHA256Hash(passphrase)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
		// panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		// log.Fatal(err)
		// panic(err.Error())
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func DecryptData(data []byte, passphrase string) ([]byte, error) {
	// key := createHash(passphrase)
	key := createSHA256Hash(passphrase)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		// panic(err.Error())
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		// panic(err.Error())
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// log.Fatal(err)
		// panic(err.Error())
		return nil, err
	}
	return plaintext, nil
}

func encryptFile(filename string, data []byte, passphrase string) {
	f, _ := os.Create(filename)
	defer f.Close()
	buf, _ := EncryptData(data, passphrase)
	f.Write(buf)
}

func decryptFile(filename string, passphrase string) []byte {
	data, _ := os.ReadFile(filename)
	buf, _ := DecryptData(data, passphrase)
	return buf
}

func CTTest() {
	fmt.Println("Starting the application...")
	ciphertext, _ := EncryptData([]byte("Hello World"), "password")
	fmt.Printf("Encrypted: %x\n", ciphertext)

	plaintext, _ := DecryptData(ciphertext, "password")
	fmt.Printf("Decrypted: %s\n", plaintext)

	encryptFile("sample.txt", []byte("Hello World"), "password1")
	fmt.Println("read from file: ", string(decryptFile("sample.txt", "password1")))

}
func CT() {
	body, _ := os.ReadFile("gdms.pem")

	cpt, _ := EncryptData(body, "rmtssh")
	fmt.Printf("encrypted: %x\n", cpt)

	plaintext, _ := DecryptData(cpt, "rmtssh")
	fmt.Printf("decrypted: %s\n", plaintext)

}
