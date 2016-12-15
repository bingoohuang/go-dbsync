package main

import (
	"os/exec"
	"log"
	"bufio"
	"github.com/valyala/gorpc"
	"os"
	"encoding/json"
	"./logrpc"
)

func main() {
	logFile := os.Args[1]
	logName := os.Args[2]
	cmd := exec.Command("tail", "-F", logFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdoutReader := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		log.Fatal("Buffer Error:", err)
	}

	c := &gorpc.Client{
		// TCP address of the server. rpc.server.addr
		Addr: "go.log.server:10811",
	}
	c.Start()

	hostName, _ := os.Hostname()

	for {
		str, err := stdoutReader.ReadString('\n')
		if err != nil {
			log.Fatal("Read Error:", err)
			return
		}

		msg := logrpc.Message{
			OriginalLog: logFile,
			LogName: logName,
			Hostname: hostName,
			Body : str,
		}

		jsonBytes, _ := json.Marshal(msg)

		c.Call(jsonBytes)
	}
}
