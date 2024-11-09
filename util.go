package main

import "encoding/json"

func MapToStruct(m map[string]interface{}, s interface{}) error {
	j, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, s)
}

func IsRepeatedChar(s string) bool {
	if s == "" {
		return false
	}
	cs := []rune(s)
	char := cs[0]

	for i := 1; i < len(cs); i++ {
		if cs[i] != char {
			return false
		}
	}
	return true
}
