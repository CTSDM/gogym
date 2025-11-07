package util

import (
	"errors"
	"fmt"
	"time"
)

var ErrMaxMinIncoherent = errors.New("min and max values are swapped")

func ValidateString(value string, min, max int) error {
	if min > max {
		return ErrMaxMinIncoherent
	}
	if len(value) > max || len(value) < min {
		errMsgFormat := "length must be greather than %d and less %d characters long"
		return fmt.Errorf(errMsgFormat, min, max)
	}
	return nil
}

func ValidateDate(value, format string, min, max *time.Time) (time.Time, error) {
	date, err := time.Parse(format, value)
	if err != nil {
		return time.Time{}, err
	}

	errMsg := ""

	if min != nil && date.Before(*min) {
		errMsg = "must be after " + min.Format(format)
	} else if max != nil && date.After(*max) {
		errMsg = "must be before " + max.Format(format)
	} else if min != nil && max != nil && !date.After(*max) && !date.Before(*min) {
		return date, nil
	}

	if errMsg != "" {
		return time.Time{}, errors.New(errMsg)
	}

	return date, nil
}
