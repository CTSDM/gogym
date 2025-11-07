package apiconstants

import "time"

const (
	DATE_LAYOUT          string = "2006-01-02"
	DATE_TIME_LAYOUT     string = "2006-01-02-150405"
	MinUsernameLength           = 4
	MaxUsernameLength           = 16
	MinPasswordLength           = 8
	MaxPasswordLength           = 256
	MinCountryLength            = 4
	MaxCountryLength            = 100
	MinSessionNameLength        = 1
	MaxSessionNameLength        = 100
	MaxRestTimeSeconds          = 3600
	MaxExerciseLength           = 200
	MaxDescriptionLength        = 500
)

var (
	MinBirthDate = time.Date(1905, time.January, 1, 0, 0, 0, 0, time.UTC)
	MaxBirthDate = time.Date(2012, time.January, 1, 0, 0, 0, 0, time.UTC)
)
