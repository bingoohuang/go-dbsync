package myutil

import "bytes"

type MyStr struct {
	buf bytes.Buffer
}

func (my *MyStr) PS(s string) *MyStr {
	my.buf.WriteString(s)
	return my
}

func (my *MyStr) PM(m map[string]string) *MyStr {
	for key, val := range m {
		my.PKV(key, val)
	}

	return my
}

func (my *MyStr) PKV(key, val string) *MyStr {
	if my.buf.Len() > 1 {
		my.PS(", ")
	}

	my.PS(key).PS(":").PS(val)

	return my
}

func (my *MyStr) Str() string {
	return my.buf.String()
}

func (my *MyStr) ReplaceLast(s string) *MyStr {
	my.buf.Truncate(my.buf.Len() - len(s))
	my.PS(s)
	return my
}
