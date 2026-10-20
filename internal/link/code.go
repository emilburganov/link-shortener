package link

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateCode(n int) (string, error) {
	limit := big.NewInt(int64(len(alphabet)))
	code := make([]byte, n)

	for i := range code {
		idx, err := rand.Int(rand.Reader, limit)

		if err != nil {
			return "", fmt.Errorf("generate code: %w", err)
		}

		code[i] = alphabet[idx.Int64()]
	}

	return string(code), nil
}
