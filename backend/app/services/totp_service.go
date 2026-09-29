package services

import (
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPService struct{}

func NewTOTPService() *TOTPService {
	return &TOTPService{}
}

func (s *TOTPService) GenerateSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Basic App",
		AccountName: "test",
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", err
	}

	return key.Secret(), nil
}

func (s *TOTPService) VerifyCode(
	secret string,
	code string,
) error {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)

	if secret == "" || code == "" {
		return ErrInvalidTOTPCode
	}

	valid, err := totp.ValidateCustom(
		code,
		secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		},
	)

	// 	| Skew | Accepted windows | Approx. tolerance |
	// |---:|---|---:|
	// | `0` | Current window only | ~30 sec |
	// | `1` | Previous + current + next | ~90 sec total |
	// | `2` | 2 previous + current + 2 next | ~150 sec |
	// | `3` | 3 previous + current + 3 next | ~210 sec |

	if err != nil {
		return err
	}

	if !valid {
		return ErrInvalidTOTPCode
	}

	return nil
}
