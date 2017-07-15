package main

import (
	"../logrpc"
	"../myutil"
	"bufio"
	"encoding/json"
	"flag"
	"github.com/valyala/gorpc"
	"log"
	"os"
	"os/exec"
	"strconv"
)

var (
	logItems      []myutil.LogItem
	logServer     string
	logServerPort int
)

func init() {
	logFlag := flag.String("log", "", "tail log file path")
	logServerArg := flag.String("logServer", "go.log.server", "log server address")
	logServerPortArg := flag.Int("port", 10811, "log server port")

	flag.Parse()

	logItems = myutil.ParseLogItems(*logFlag)
	logServer = *logServerArg
	logServerPort = *logServerPortArg
}

func main() {
	msgChan := make(chan logrpc.Message)

	for _, logItem := range logItems {
		go tail(logItem, msgChan)
	}

	writeToServer(msgChan)

}

func writeToServer(msgChan chan logrpc.Message) {
	c := &gorpc.Client{
		Addr: logServer + ":" + strconv.Itoa(logServerPort),
	}
	c.Start()

	for msg := range msgChan {
		jsonBytes, _ := json.Marshal(msg)
		c.Call(jsonBytes)
	}
}

func tail(item myutil.LogItem, msgChan chan logrpc.Message) {
	hostName, _ := os.Hostname()
	cmd := exec.Command("tail", "-F", item.LogFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdoutReader := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		log.Fatal("Buffer Error:", err)
	}

	for {
		str, err := stdoutReader.ReadString('\n')
		if err != nil {
			log.Fatal("Read Error:", err)
			return
		}

		msgChan <- logrpc.Message{
			OriginalLog: item.LogFile,
			LogName:     item.LogName,
			Hostname:    hostName,
			Body:        str,
		}
	}
}
