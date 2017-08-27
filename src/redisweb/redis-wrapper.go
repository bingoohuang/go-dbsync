package main

import (
	"errors"
	"github.com/go-redis/redis"
)

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

type ContentResult struct {
	Content  string
	Ttl      string
	Encoding string
	Size     int64
}

func displayContent(key string, valType string) (*ContentResult, error) {
	client := newRedisClient()
	defer client.Close()

	switch valType {
	case "string":
		content, _ := client.Get(key).Result()
		ttl, _ := client.TTL(key).Result()
		size, _ := client.StrLen(key).Result()
		encoding, _ := client.ObjectEncoding(key).Result()
		return &ContentResult{
			Content:  content,
			Ttl:      ttl.String(),
			Encoding: encoding,
			Size:     size,
		}, nil
	}

	return nil, errors.New("unknown type " + valType)

}

type KeysResult struct {
	Key  string
	Type string
	Len  int64
}

func listKeys(matchPattern string, maxKeys int) ([]KeysResult, error) {
	client := newRedisClient()
	defer client.Close()

	allKeys := make([]KeysResult, 0)
	var cursor uint64
	for {
		keys, cursor, err := client.Scan(cursor, matchPattern, 10).Result()
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
