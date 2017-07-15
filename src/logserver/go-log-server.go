package main

import (
	"../logrpc"
	"encoding/json"
	"github.com/valyala/gorpc"
	"log"
	"os"
	"flag"
	"strconv"
)

var port int

func init() {
	portArg := flag.Int("port", 10811, "log server port")
	flag.Parse()

	port = *portArg
}

func main() {
	msgChan := make(chan string)
	s := &gorpc.Server{
		Addr: ":" + strconv.Itoa(port),

		Handler: func(clientAddr string, request interface{}) interface{} {
			msgChan <- string(request.([]byte))
			return "ok"
		},
	}

	go func() {
		if err := s.Serve(); err != nil {
			log.Fatalf("Cannot start rpc server: %s", err)
		}
	}()

	logfileMap := make(map[string]*os.File)
	for msg := range msgChan {
		msg := parseMessage(msg)

		logFile := getOrCreateFile(logfileMap, msg.LogName)
		logFile.WriteString(msg.Body)
		logFile.Sync()
	}
}

func parseMessage(msg string) logrpc.Message {
	var f logrpc.Message
	err := json.Unmarshal([]byte(msg), &f)
	if err != nil {
		log.Fatalf("Cannot start rpc server: %s", err)
	}
	return f
}

func getOrCreateFile(logfileMap map[string]*os.File, logName string) *os.File {
	logFile, ok := logfileMap[logName]
	if ok {
		return logFile
	}

	flag := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	logFile, _ = os.OpenFile(logName, flag, 0600)
	logfileMap[logName] = logFile

	return logFile
}
