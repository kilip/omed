package util

import "encoding/json"

func ToValue[T any](source any, target *T) error {
	jsonData, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}
