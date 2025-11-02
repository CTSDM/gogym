package api

import (
	"errors"
	"fmt"
	"time"
)

const (
	minUsernameLength = 4
	maxUsernameLength = 16
	minPasswordLength = 8
	maxPasswordLength = 256
)

var (
	minBirthDate = time.Date(1905, time.January, 1, 0, 0, 0, 0, time.UTC)
	maxBirthDate = time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)
)

var ErrMaxMinIncoherent = errors.New("min and max values are swapped")

func validateString(value string, min, max int) error {
	if min > max {
		return ErrMaxMinIncoherent
	}
	if len(value) > max || len(value) < min {
		errMsgFormat := "length must be greather than %d and less %d characters long"
		return fmt.Errorf(errMsgFormat, min, max)
	}
	return nil
}

func validateDate(value, format string, min, max *time.Time) error {
	date, err := time.Parse(format, value)
	if err != nil {
		return err
	}

	errMsg := ""

	if min != nil && date.Before(*min) {
		errMsg = "must be after " + min.Format(format)
	} else if max != nil && date.After(*max) {
		errMsg = "must be before " + max.Format(format)
	} else if min != nil && max != nil && !date.After(*max) && !date.Before(*min) {
		return nil
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}
