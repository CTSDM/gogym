package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// taken from Mat Ryer's article
// https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
func DecodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}

type Validator interface {
	Valid(ctx context.Context) (problems map[string]string)
}

var ErrMaxMinIncoherent = errors.New("min and max values are swapped")

func String(value string, min, max int) error {
	if min > max {
		return ErrMaxMinIncoherent
	}
	if len(value) > max || len(value) < min {
		errMsgFormat := "length must be greather than %d and less %d characters long"
		return fmt.Errorf(errMsgFormat, min, max)
	}
	return nil
}

func Date(value, format string, min, max *time.Time) (time.Time, error) {
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
