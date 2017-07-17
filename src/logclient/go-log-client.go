package main

import (
	"../logrpc"
	"../myutil"
	"encoding/json"
	"flag"
	"github.com/valyala/gorpc"
	"log"
	"os"
	"os/exec"
	"strconv"
	"io"
	"bufio"
)

var (
	logItems      []myutil.LogItem
	logServer     string
	logServerPort int
)

func init() {
	logFlag := flag.String("log", "", "tail log file path")
	serverArg := flag.String("server", "go.log.server", "log server address")
	portArg := flag.Int("port", 10811, "log server port")

	flag.Parse()

	logItems = myutil.ParseLogItems(*logFlag)
	logServer = *serverArg
	logServerPort = *portArg
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
	defer stdout.Close()

	reader := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		log.Fatal("Buffer Error:", err)
	}

	dataChan := make(chan []byte)
	go readInput(reader, dataChan)

	for msg := range dataChan {
		msgChan <- logrpc.Message{
			OriginalLog: item.LogFile,
			LogName:     item.LogName,
			Hostname:    hostName,
			Body:        string(msg),
		}
	}
}

func readInput(reader *bufio.Reader, dataChan chan []byte) {
	tmp := make([]byte, 10240)
	for {
		length, err := reader.Read(tmp)
		if err != nil && err != io.EOF {
			log.Println("read error", err.Error())
			break
		}

		dataChan <- tmp[0:length]
	}
}
