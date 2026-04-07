package transport

import "encoding/json"

func Encode[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

func Decode[T any](raw []byte, target *T) error {
	return json.Unmarshal(raw, target)
}
