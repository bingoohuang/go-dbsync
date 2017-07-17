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

var (
	port int
)

func init() {
	portArg := flag.Int("port", 10811, "log server port")
	flag.Parse()

	port = *portArg
}

func main() {
	msgChan := make(chan []byte)

	go startServer(msgChan)

	logfileMap := make(map[string]*os.File)
	for rpcMsg := range msgChan {
		msg := parseMessage(rpcMsg)

		logFile := getOrCreateFile(logfileMap, msg.LogName)
		logFile.WriteString(msg.Body)
		logFile.Sync()
	}
}

func startServer(msgChan chan []byte) {
	s := &gorpc.Server{
		Addr: ":" + strconv.Itoa(port),

		Handler: func(clientAddr string, request interface{}) interface{} {
			msgChan <- request.([]byte)
			return "ok"
		},
	}

	if err := s.Serve(); err != nil {
		log.Fatalf("Cannot start rpc server: %s", err)
	}
}

func parseMessage(rpcMsg []byte) logrpc.Message {
	var msg logrpc.Message
	err := json.Unmarshal(rpcMsg, &msg)
	if err != nil {
		log.Fatalf("Cannot start rpc server: %s", err)
	}
	return msg
}

func getOrCreateFile(logfileMap map[string]*os.File, logName string) *os.File {
	logFile, ok := logfileMap[logName]
	if ok {
		return logFile
	}

	flag := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	logFile, _ = os.OpenFile(logName + ".log", flag, 0600)
	logfileMap[logName] = logFile

	return logFile
}
