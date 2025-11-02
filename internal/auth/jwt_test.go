package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMakeRefreshToken(t *testing.T) {
	t.Run("must generate random strings", func(t *testing.T) {
		tokens := make(map[string]struct{}, 0)
		iterations := 1_000
		for range iterations {
			token, err := MakeRefreshToken()
			assert.NoError(t, err)
			tokens[token] = struct{}{}
		}
		assert.Equal(t, iterations, len(tokens), "there was collision of token")
	})
}

func TestMakeJWT(t *testing.T) {
	testCases := []struct {
		id        string
		secret    string
		expiresIn time.Duration
		hasError  bool
	}{
		{
			id:        "randomstring",
			secret:    "secret",
			expiresIn: time.Hour,
		},
		{
			id:        "randomstring",
			secret:    "secret",
			expiresIn: 0,
			hasError:  true,
		},
		{
			id:        "randomstring",
			secret:    "",
			expiresIn: time.Hour,
			hasError:  true,
		},
		{
			id:        "",
			secret:    "somesecret",
			expiresIn: time.Hour,
			hasError:  true,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"test number %d - secret '%s', expires in %s, should fail %t",
				i,
				tc.secret,
				tc.expiresIn.String(),
				tc.hasError),
			func(t *testing.T) {
				secret := tc.secret
				token, err := MakeJWT(tc.id, secret, tc.expiresIn)
				if tc.hasError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotEqual(t, "", token, "generated token must not be empty")
			})
	}
}

func TestValidateJWT(t *testing.T) {
	testCases := []struct {
		token          string
		secret         string
		subject        string
		duration       time.Duration
		shouldMatch    bool
		hasWrongSecret bool
		hasError       bool
	}{
		{
			secret:      "secret",
			subject:     "gogymgogym",
			duration:    time.Hour,
			shouldMatch: true,
		},
		{
			secret:      "secret",
			subject:     "gogymgogym",
			duration:    time.Hour,
			shouldMatch: false,
		},
		{
			secret:         "wrong_secret",
			subject:        "gogymgogym",
			duration:       time.Hour,
			hasWrongSecret: true,
			hasError:       true,
		},
		{
			token:    "not_a_token",
			subject:  "gogymgogym",
			secret:   "not_a_secret",
			duration: time.Hour,
			hasError: false,
		},
		{
			secret:   "secret",
			subject:  "gogymgogym",
			duration: 1 * time.Microsecond,
			hasError: true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test number %d, should work: %t, ", i, !tc.hasError), func(t *testing.T) {
			// create the JWT
			token, err := MakeJWT(tc.subject, tc.secret, tc.duration)
			assert.NoError(t, err, "did not expect an error while creating JWT")
			// change the secret if needed
			secret := tc.secret
			if tc.hasWrongSecret == true {
				secret += secret
			}
			got, err := ValidateJWT(token, secret)
			if tc.hasError == true {
				assert.Error(t, err)
				return
			}
			if tc.shouldMatch == true {
				assert.Equal(t, tc.subject, got)
			} else {
				got, err := MakeJWT(tc.subject+"test", tc.secret, tc.duration)
				assert.NoError(t, err, "did not expect an error while creating JWT")
				assert.NotEqual(t, tc.subject, got)
			}
		})
	}
}
