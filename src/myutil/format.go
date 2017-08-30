package myutil

import (
	"encoding/json"
	"unicode"
)

func IsJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil && s != "" && (s[0] == '{' || s[0] == '[')
}

func IsPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
