package std

import "errors"

var (
	ErrInvalidSetParams = errors.New("err: invalid params to generate circuit")
	ErrInvalidRangeParams      = errors.New("err: invalid params for range proof")
)
