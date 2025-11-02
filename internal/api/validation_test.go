package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateString(t *testing.T) {
	t.Run("Check with coherent limits", func(t *testing.T) {
		testCases := []struct {
			value    string
			min      int
			max      int
			hasError bool
		}{
			{
				value: "works",
				min:   1,
				max:   10,
			},
			{
				value: "four",
				min:   4,
				max:   4,
			},
			{
				value: "verylongpassword",
				min:   0,
				max:   100,
			},
			{
				value:    "",
				min:      5,
				max:      10,
				hasError: true,
			},
			{
				value:    "hello",
				max:      1,
				hasError: true,
			},
			{
				value:    "inverse limits",
				min:      10,
				max:      0,
				hasError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("value %s, min length %d, max length %d", tc.value, tc.min, tc.max), func(t *testing.T) {
				err := validateString(tc.value, tc.min, tc.max)
				if tc.hasError == true {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
	t.Run("Check incoherent limits", func(t *testing.T) {
		min := 10
		max := 0
		err := validateString("", min, max)
		assert.EqualError(t, err, ErrMaxMinIncoherent.Error())
	})
}
