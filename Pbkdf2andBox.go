package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/pbkdf2"
)

type Service interface {
	EncryptWithPbkdf2(data []byte) ([]byte, error)
	DecryptWithPbkdf2GCM(data []byte) ([]byte, error)
	DecryptWithPbkdf2CBC(data []byte) ([]byte, error)
	DecryptBox(key, encryptedData string) ([]byte, error)
}

type service struct {
	secret []byte
	iter   int
}

func NewCryptoService(secret *string, iter *int) (Service, error) {
	if secret == nil {
		return nil, errors.New("invalid crypto secret")
	}
	if iter == nil {
		return nil, errors.New("invalid crypto iter")
	}

	return &service{
		secret: []byte(*secret),
		iter:   *iter,
	}, nil
}

func (s *service) DecryptBox(key, encryptedData string) ([]byte, error) {
	// Setup key
	var k [32]byte
	if n, err := hex.Decode(k[:], []byte(key)); err != nil || n != len(k) {
		return nil, err
	}

	// Setup data
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	// Setup nonce
	var decryptNonce [24]byte
	copy(decryptNonce[:], data[:24])

	dec, ok := box.OpenAfterPrecomputation(nil, data[24:], &decryptNonce, &k)

	if !ok {
		return nil, err
	}

	return dec, nil
}

func (s *service) EncryptWithPbkdf2(data []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key, err := s.deriveKey(salt)
	if err != nil {
		return nil, err
	}

	blocCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gmc, err := cipher.NewGCM(blocCipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gmc.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	cipherText := gmc.Seal(nonce, nonce, data, nil)

	cipherText = append(cipherText, salt...)

	return cipherText, nil
}

func (s *service) DecryptWithPbkdf2GCM(data []byte) ([]byte, error) {
	salt, data := data[len(data)-32:], data[:len(data)-32]

	key, err := s.deriveKey(salt)
	if err != nil {
		return nil, err
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gmc, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}

	nonce, cipherText := data[:gmc.NonceSize()], data[gmc.NonceSize():]

	plainText, err := gmc.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plainText, nil
}

func (s *service) DecryptWithPbkdf2CBC(data []byte) ([]byte, error) {
	salt, data := data[:32], data[32:]

	key, err := s.deriveKey(salt)
	if err != nil {
		return nil, err
	}

	nonce, cipherText := data[:16], data[16:]

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(blockCipher, nonce)
	if err != nil {
		return nil, err
	}
	mode.CryptBlocks(cipherText, cipherText)

	cipherText, _ = s.unpad(cipherText, aes.BlockSize)

	return []byte(cipherText), nil
}

func (s *service) deriveKey(salt []byte) ([]byte, error) {
	key := pbkdf2.Key(s.secret, salt, s.iter, 32, sha256.New)

	return key, nil
}

func (s *service) unpad(padded []byte, size int) ([]byte, error) {
	if len(padded)%size != 0 {
		return nil, errors.New("Padded value wasn't in correct size.")
	}

	bufLen := len(padded) - int(padded[len(padded)-1])
	buf := make([]byte, bufLen)
	copy(buf, padded[:bufLen])
	return buf, nil
}

func main() {
	sec := "asd123"
	iter := 100

	cryptoSvc, err := NewCryptoService(&sec, &iter)
	if err != nil {
		log.Fatal(err)
	}

	// chipText, err := cryptoSvc.EncryptWithPbkdf2([]byte("aaaaaaa"))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// encodeText := base64.StdEncoding.EncodeToString(chipText)
	// fmt.Println(encodeText)

	decodeBase64, err := base64.StdEncoding.DecodeString("va2QfnKDbj+LCyjQNf4RE1eLDWw5Z/QDR2Y2tzEGzCB3gMyfIZ2QYlzKcWDhVHaUopdta/MuktDtE/n0Tv3Z4w==")
	if err != nil {
		log.Fatal(err)
	}

	decodeText, err := cryptoSvc.DecryptWithPbkdf2CBC(decodeBase64)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(decodeText))
}
