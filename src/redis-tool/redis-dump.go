package main

import (
	"fmt"
	"strconv"
	"github.com/go-redis/redis"
	"strings"
	"flag"
)

var (
	redisHost string
	redisAuth string
	redisPort int
	redisDb   int
)

func init() {
	redisHostArg := flag.String("ip", "127.0.0.1", "redis-tool host")
	redisAuthArg := flag.String("auth", "", "redis-tool auth")
	redisPortArg := flag.Int("port", 6379, "redis-tool port")
	redisDbArg := flag.Int("db", 0, "redis-tool db index")

	flag.Parse()

	redisHost = *redisHostArg
	redisAuth = *redisAuthArg
	redisPort = *redisPortArg
	redisDb = *redisDbArg
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + strconv.Itoa(redisPort),
		Password: redisAuth, // no password set
		DB:       redisDb,   // use default DB
	})
	defer client.Close()

	keys, err := client.Keys("*").Result()
	if err != nil {
		println("Redis-dump failed", err.Error())
		return
	}

	for _, key := range keys {
		typ, _ := client.Type(key).Result()

		switch typ {
		case "string":
			data, _ := client.Get(key).Result()
			fmt.Println(key, ":", data)
		case "list":
			data, _ := client.LRange(key, 0, -1).Result()
			fmt.Println(key, ":", strings.Join(data, ", "))
		case "set":
			data, _ := client.SMembers(key).Result()
			fmt.Println(key, ":", strings.Join(data, ", "))
		case "sortedset":
			data, _ := client.ZRange(key, 0, -1).Result()
			fmt.Println(key, ":", strings.Join(data, ", "))
		default:
			fmt.Println(key, ": unkown type ", typ)
		}
	}

}
