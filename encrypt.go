package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

func ParseEncrypt(body, baseUrlPrefix string) Encrypt {
	var encrypt Encrypt
	for _, v := range strings.Split(body, "\n") {
		prefix := "#EXT-X-KEY:"
		if strings.Contains(v, prefix) {
			v = v[len(prefix):]
			tagParam := make(map[string]string)
			for _, pair := range strings.Split(v, ",") {
				kv := strings.Split(pair, "=")
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.Trim(kv[1], `"`)
					tagParam[key] = value
				}
			}
			log.Printf("key: %v", tagParam)
			switch tagParam["METHOD"] {
			case "AES-128":
				encrypt.method = tagParam["METHOD"]
				encrypt.uri = tagParam["URI"]
				ivBytes, err := hex.DecodeString(strings.TrimPrefix(tagParam["IV"], "0x"))
				if err != nil {
					log.Fatalf("无法解码IV: %v", err)
				}
				encrypt.iv = ivBytes
				key, err := HttpGet(&HttpRequestConfig{
					URL: fmt.Sprintf("%s/%s", baseUrlPrefix, encrypt.uri),
				})
				if err != nil {
					log.Fatal(err)
				}
				encrypt.key = key
			case "NONE":
				encrypt.method = tagParam["METHOD"]
			}
		}
	}
	return encrypt
}

func AESDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	paddingLen := int(plaintext[len(plaintext)-1])
	return plaintext[:len(plaintext)-paddingLen], nil
}
