package commonerror

import "fmt"

type ApiError struct {
	StatusCode int
	Code       string
	Message    string
}

func (err ApiError) Error() string {
	return fmt.Sprintf("ApiError: %s", err.Message)
}
