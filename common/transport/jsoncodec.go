package transport

import "encoding/json"

type validator interface {
	Validate() error
}

func Encode[T any](value T) ([]byte, error) {
	if v, ok := any(value).(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}

	return json.Marshal(value)
}

func Decode[T any](raw []byte, target *T) error {
	return json.Unmarshal(raw, target)
}
