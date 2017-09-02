package main

import (
	"../myutil"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type RedisServer struct {
	ServerName string
	Addr       string
	Password   string
	DB         int
}

var (
	contextPath string
	port        int

	devMode bool // to disable css/js minify
	servers []RedisServer
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 8269, "Port to serve.")
	devModeArg := flag.Bool("devMode", false, "devMode(disable js/css minify)")
	serversArg := flag.String("servers", "default=localhost:6379", "servers list, eg: Server1=localhost:6379,Server2=password2/localhost:6388/0")

	flag.Parse()

	contextPath = *contextPathArg
	port = *portArg
	devMode = *devModeArg
	servers = parseServers(*serversArg)
}

func parseServers(serversConfig string) []RedisServer {
	serverItems := myutil.SplitTrim(serversConfig, ",")

	var result = make([]RedisServer, 0)
	for i, item := range serverItems {
		parts := myutil.SplitTrim(item, "=")
		len := len(parts)
		if len == 1 {
			serverName := "Server" + strconv.Itoa(i+1)
			result = append(result, parseServerItem(serverName, parts[0]))
		} else if len == 2 {
			serverName := parts[0]
			result = append(result, parseServerItem(serverName, parts[1]))
		} else {
			panic("invalid servers argument")
		}
	}

	return result
}
func parseServerItem(serverName, serverConfig string) RedisServer {
	serverItems := myutil.SplitTrim(serverConfig, "/")
	len := len(serverItems)
	if len == 1 {
		return RedisServer{
			ServerName: serverName,
			Addr:       serverItems[0],
			Password:   "",
			DB:         0,
		}
	} else if len == 2 {
		dbIndex, _ := strconv.Atoi(serverItems[1])
		return RedisServer{
			ServerName: serverName,
			Addr:       serverItems[0],
			Password:   "",
			DB:         dbIndex,
		}
	} else if len == 3 {
		dbIndex, _ := strconv.Atoi(serverItems[2])
		return RedisServer{
			ServerName: serverName,
			Addr:       serverItems[1],
			Password:   serverItems[0],
			DB:         dbIndex,
		}
	} else {
		panic("invalid servers argument")
	}
}

func main() {
	http.HandleFunc(contextPath+"/", myutil.GzipWrapper(serveHome))
	http.HandleFunc(contextPath+"/favicon.png", serveImage("favicon.png"))
	http.HandleFunc(contextPath+"/spritesheet.png", serveImage("spritesheet.png"))
	http.HandleFunc(contextPath+"/listKeys", serveListKeys)
	http.HandleFunc(contextPath+"/showContent", serveShowContent)
	http.HandleFunc(contextPath+"/changeContent", serveChangeContent)
	http.HandleFunc(contextPath+"/deleteKey", serveDeleteKey)
	http.HandleFunc(contextPath+"/newKey", serveNewKey)
	http.HandleFunc(contextPath+"/redisInfo", serveRedisInfo)

	sport := strconv.Itoa(port)
	fmt.Println("start to listen at ", sport)
	if err := http.ListenAndServe(":"+sport, nil); err != nil {
		log.Fatal(err)
	}
}
