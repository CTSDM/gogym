package auth

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("Hashed password should be different from password", func(t *testing.T) {
		passwords := []string{"password123", "test123"}
		for _, password := range passwords {
			t.Run(password, func(t *testing.T) {
				hashed, err := HashPassword(password)
				assert.NoError(t, err, "could not hash password")
				assert.NotEqual(t, password, hashed, "password and hashed password should not match")
			})
		}
	})

	t.Run("Empty strings should work too", func(t *testing.T) {
		password := ""
		hashed, err := HashPassword(password)
		assert.NoError(t, err, "could not hash password")
		assert.NotEqual(t, password, hashed, "expected empty password to produce a hash")
	})

	t.Run("Same password should generate different hashes", func(t *testing.T) {
		password := "test123"
		hashed1, err1 := HashPassword(password)
		hashed2, err2 := HashPassword(password)
		assert.NoError(t, err1, "could not hash password first time")
		assert.NoError(t, err2, "could not hash password second time")
		assert.False(t, hashed1 == hashed2, "same cleartext password should produce different hashes")
	})

	t.Run("Testing maximum password length", func(t *testing.T) {
		testCases := []struct {
			passwordLength int
			err            error
		}{
			{
				passwordLength: 12,
				err:            nil,
			},
			{
				passwordLength: 40,
				err:            nil,
			},
			{
				passwordLength: 72,
				err:            nil,
			},
			{
				passwordLength: 73,
				err:            bcrypt.ErrPasswordTooLong,
			},
		}

		wg := sync.WaitGroup{}
		for _, tc := range testCases {
			wg.Go(func() {
				t.Run(fmt.Sprintf("%d", tc.passwordLength), func(t *testing.T) {
					randomPassword := make([]byte, tc.passwordLength)
					_, err := rand.Read(randomPassword)
					assert.NoError(t, err, "could not generate the random password")
					_, err = HashPassword(string(randomPassword))
					assert.ErrorIs(t, err, tc.err)
				})
			})
		}
		wg.Wait()
	})

	t.Run("Generates a valid bcrypt hash", func(t *testing.T) {
		hashed, err := HashPassword("test123")
		assert.NoError(t, err, "could not hash password")
		cost, err := bcrypt.Cost([]byte(hashed))
		assert.NoError(t, err, fmt.Sprintf("invalid bcrypt format: %s", hashed))
		assert.Equal(t, cost, COST_HASHING, "cost hashing should match")
	})
}

func TestCheckPasswordHash(t *testing.T) {
	testCases := []struct {
		password         string
		hashed           string
		generatePassword bool
		hasError         bool
	}{
		{
			password:         "passwordtest",
			generatePassword: true,
			hasError:         false,
		},
		{
			password:         "anotherpassword",
			hashed:           "random",
			generatePassword: false,
			hasError:         true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("the password and hash should match: %t", tc.generatePassword), func(t *testing.T) {
			if tc.generatePassword == true {
				hashed, err := HashPassword(tc.password)
				assert.NoError(t, err, "could not generate the password")
				tc.hashed = hashed
			}
			err := CheckPasswordHash(tc.password, tc.hashed)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
