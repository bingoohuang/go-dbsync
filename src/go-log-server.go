package main

import (
	"log"
	"./logrpc"
	"github.com/valyala/gorpc"
	"encoding/json"
	"os"
)

func main() {
	msgChan := make(chan string)
	s := &gorpc.Server{
		Addr: ":10811",

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
		var f logrpc.Message
		err := json.Unmarshal([]byte(msg), &f)
		if err != nil {
			log.Fatalf("Cannot start rpc server: %s", err)
		}

		logFile, ok := logfileMap[f.LogName]
		if !ok {
			flag := os.O_CREATE | os.O_APPEND | os.O_WRONLY
			logFile, _ := os.OpenFile(f.LogName, flag, 0600)
			logfileMap[f.LogName] = logFile
		}

		logFile.WriteString(msg + "\n");
		logFile.Sync()
	}
}
