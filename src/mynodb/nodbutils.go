package mynodb

import (
	"fmt"
	"github.com/lunny/nodb"
	"github.com/lunny/nodb/config"
	"io/ioutil"
	"os"
)

type Nodb struct {
	db *nodb.DB
}

func (db *Nodb) Set(key, value string) error {
	return db.db.Set([]byte(key), []byte(value))
}

func (db *Nodb) Get(key string) (string, error) {
	value, err := db.db.Get([]byte(key))
	str := string(value)
	return str, err
}

func (db *Nodb) Exists(key string) bool {
	value, _ := db.db.Exists([]byte(key))
	return value == 1
}

func OpenTemp() (*Nodb, string, error) {
	cfg := new(config.Config)

	cfg.DataDir, _ = ioutil.TempDir(os.TempDir(), "nodb")
	nodbs, err := nodb.Open(cfg)
	if err != nil {
		fmt.Printf("nodb: error opening db: %v", err)
	}

	db, err := nodbs.Select(0)

	return &Nodb{db}, cfg.DataDir, err
}
