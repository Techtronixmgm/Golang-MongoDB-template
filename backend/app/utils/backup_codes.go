package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const backupCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateBackupCodes(count int) ([]string, []string, error) {
	if count <= 0 {
		return nil, nil, errors.New(
			"backup code count must be greater than zero",
		)
	}

	backupCodes := make([]string, 0, count)
	backupCodeHashes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, nil, err
		}

		hash, err := bcrypt.GenerateFromPassword(
			[]byte(code),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, nil, err
		}

		backupCodes = append(backupCodes, code)
		backupCodeHashes = append(
			backupCodeHashes,
			string(hash),
		)
	}

	return backupCodes, backupCodeHashes, nil
}

func generateBackupCode() (string, error) {
	const codeLength = 8

	characters := make([]byte, codeLength)

	for i := range characters {
		index, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(backupCodeAlphabet))),
		)
		if err != nil {
			return "", err
		}

		characters[i] = backupCodeAlphabet[index.Int64()]
	}

	return fmt.Sprintf(
		"%s-%s",
		string(characters[:4]),
		string(characters[4:]),
	), nil
}
