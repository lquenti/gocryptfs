// +build !without_openssl

package stupidgcm

// #include <openssl/evp.h>
import "C"

import (
	"crypto/cipher"
	"log"
)

const (
	// BuiltWithoutOpenssl indicates if openssl been disabled at compile-time
	BuiltWithoutOpenssl = false

	keyLen = 32
	ivLen  = 16
	tagLen = 16
)

type stupidGCM struct {
	stupidAEADCommon
}

// New returns a new AES-GCM-256 cipher that satisfies the cipher.AEAD interface.
//
// Only 32-bytes keys and 16-byte IVs are supported.
func New(keyIn []byte, forceDecode bool) cipher.AEAD {
	if len(keyIn) != keyLen {
		log.Panicf("Only %d-byte keys are supported", keyLen)
	}
	return &stupidGCM{
		stupidAEADCommon{
			// Create a private copy of the key
			key:              append([]byte{}, keyIn...),
			openSSLEVPCipher: C.EVP_aes_256_gcm(),
			nonceSize:        ivLen,
		},
	}
}
