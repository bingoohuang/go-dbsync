package main

import (
	"bytes"
	"io"
	"math/rand"
	"time"
)

func readObjectString(object io.ReadCloser) string {
	defer object.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	return buf.String()
}
func readObjectBytes(object io.ReadCloser) []byte {
	defer object.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	return buf.Bytes()
}

var r *rand.Rand // Rand for this package.

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandomString(size int) string {
	const chars = "23456789abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	result := ""
	for i := 0; i < size; i++ {
		index := r.Intn(len(chars))
		result += chars[index : index+1]
	}
	return result
}
