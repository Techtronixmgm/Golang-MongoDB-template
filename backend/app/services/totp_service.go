package services

import (
	"basic-app/apperrors"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPService struct {
	issuer string
}

func NewTOTPService(issuer string) *TOTPService {
	return &TOTPService{
		issuer: issuer,
	}
}

type TOTPSetup struct {
	Secret     string
	OTPAuthURL string
}

func (s *TOTPService) GenerateSecret(accountName string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}

	return &TOTPSetup{
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	}, nil
}

func (s *TOTPService) VerifyCode(
	secret string,
	code string,
) error {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)

	if secret == "" || code == "" {
		return apperrors.ErrInvalidTOTPCode
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
		return apperrors.ErrInvalidTOTPCode
	}

	return nil
}
