package main

import (
	"os"
	"time"
	"math/rand"
	"github.com/lunny/log"
)

func main() {
	flag := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	logFile, err := os.OpenFile("test.log", flag, 0600)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	rand.Seed(time.Now().UnixNano())

	for {
		randString := RandString()
		timestamp := time.Now().Format("2006-01-02 15:04:05 ")
		strLine := timestamp + randString + "\n"
		log.Print("gen:", strLine)
		if _, err = logFile.WriteString(strLine); err != nil {
			panic(err)
		}

		time.Sleep(3000 * time.Millisecond)
	}
}

func RandString() string {
	len := rand.Intn(30) + 10 // at least size 10
	bytes := make([]rune, len)
	for i := range bytes {
		//  32 Space to 126 ~ in ASCII table
		bytes[i] = rune(33 + rand.Intn(94))
	}
	return string(bytes)
}
