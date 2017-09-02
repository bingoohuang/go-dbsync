package main

import (
	"../myutil"
	"github.com/go-redis/redis"
	"strconv"
	"time"
	"strings"
)

func newRedisClient(server RedisServer) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     server.Addr,
		Password: server.Password, // no password set
		DB:       server.DB,       // use default DB
	})
}

func redisCli(server RedisServer, clientCommand string) (string, error) {
	fields := strings.Fields(clientCommand)

	client := newRedisClient(server)
	defer client.Close()

	args := make([]interface{}, len(fields))
	for index, field := range fields {
		args[index] = field
	}

	cmd := redis.NewStringCmd(args...)
	client.Process(cmd)
	return cmd.Result()
}

func redisInfo(server RedisServer) string {
	client := newRedisClient(server)
	defer client.Close()

	info, _ := client.Info().Result()
	return info
}

func configGetDatabases(server RedisServer) int {
	client := newRedisClient(server)
	defer client.Close()

	config, _ := client.ConfigGet("databases").Result()
	databaseNum, _ := strconv.Atoi(config[1].(string))
	return databaseNum
}

func changeContent(server RedisServer, key, content, format string) string {
	client := newRedisClient(server)
	defer client.Close()

	if format == "UNKNOWN" {
		content, _ = strconv.Unquote(content)
	}

	err := client.Set(key, content, 0).Err()
	if err == nil {
		return "OK"
	}

	return err.Error()
}

func newKey(server RedisServer, keyType, key, ttl, format, val string) string {
	client := newRedisClient(server)
	defer client.Close()

	var err error
	if format == "Quoted" {
		val, err = strconv.Unquote(val)
		if err != nil {
			return err.Error()
		}
	}

	var duration time.Duration = -1
	if ttl != "-1s" && ttl != "" {
		duration, err = time.ParseDuration(ttl)
		if err != nil {
			return err.Error()
		}
	}

	switch keyType {
	case "string":
		_, err = client.Set(key, val, duration).Result()

	}

	if err != nil {
		return err.Error()
	}

	return "OK"

}

func deleteKey(server RedisServer, key string) string {
	client := newRedisClient(server)
	defer client.Close()

	ok, err := client.Del(key).Result()
	if ok == 1 {
		return "OK"
	} else {
		return err.Error()
	}
}

type ContentResult struct {
	Exists   bool
	Content  string
	Ttl      string
	Encoding string
	Size     int64
	Error    string
	Format   string // JSON, NORMAL, UNKNOWN
}

func displayContent(server RedisServer, key string, valType string) *ContentResult {
	client := newRedisClient(server)
	defer client.Close()

	exists, _ := client.Exists(key).Result()
	if exists == 0 {
		return &ContentResult{
			Exists:   false,
			Content:  "",
			Ttl:      "",
			Encoding: "",
			Size:     0,
			Error:    "",
		}
	}

	var errorMessage string
	switch valType {
	case "string":

		content, err := client.Get(key).Result()
		if err != nil {
			errorMessage = err.Error()
		}
		ttl, _ := client.TTL(key).Result()
		size, _ := client.StrLen(key).Result()
		encoding, _ := client.ObjectEncoding(key).Result()
		content, format := parseFormat(content)

		return &ContentResult{
			Exists:   true,
			Content:  content,
			Ttl:      ttl.String(),
			Encoding: encoding,
			Size:     size,
			Error:    errorMessage,
			Format:   format,
		}
	}

	return &ContentResult{
		Exists:   true,
		Content:  "",
		Ttl:      "",
		Encoding: "",
		Size:     0,
		Error:    "unknown type " + valType,
		Format:   "UNKNOWN",
	}

}
func parseFormat(s string) (string, string) {
	if s == "" {
		return s, "UNKNOWN"
	}

	if myutil.IsJSON(s) {
		return myutil.JSONPrettyPrint(s), "JSON"
	}

	if myutil.IsPrintable(s) {
		return s, "NORMAL"
	}

	return strconv.Quote(s), "UNKNOWN"
}

type KeysResult struct {
	Key  string
	Type string
	Len  int64
}

func listKeys(server RedisServer, matchPattern string, maxKeys int) ([]KeysResult, error) {
	client := newRedisClient(server)
	defer client.Close()

	allKeys := make([]KeysResult, 0)
	var keys []string
	var cursor uint64
	var err error

	for {
		keys, cursor, err = client.Scan(cursor, matchPattern, 10).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			valType, err := client.Type(key).Result()
			if err != nil {
				return nil, err
			}

			var len int64
			switch valType {
			case "list":
				len, _ = client.LLen(key).Result()
			case "hash":
				len, _ = client.HLen(key).Result()
			case "set":
				len, _ = client.SCard(key).Result()
			case "zset":
				len, _ = client.ZCard(key).Result()
			default:
				len = 1
			}

			allKeys = append(allKeys, KeysResult{Key: key, Type: valType, Len: len})
		}

		if cursor == 0 || len(allKeys) >= maxKeys {
			break
		}
	}

	return allKeys, nil
}
