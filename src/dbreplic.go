package main

import (
    "github.com/jasonlvhit/gocron"
    "./myutil"
    "./mydb"
    "./mynodb"
    "os"
    "github.com/BurntSushi/toml"
    _ "github.com/go-sql-driver/mysql"
    "fmt"
)

/*
MySQL数据库单向同步.
1) 查询表名、表行数：
select TABLE_NAME,TABLE_ROWS from information_schema.tables where table_schema = DATABASE();
2) 查询列名和列类型
SELECT column_name , column_type FROM information_schema. COLUMNS
WHERE table_schema = database() AND table_name = 'cats' ORDER BY ORDINAL_POSITION;
3) 查询表主键
SELECT k.COLUMN_NAME FROM information_schema.table_constraints t LEFT JOIN information_schema.key_column_usage k USING( constraint_name , table_schema , table_name)
WHERE t.constraint_type = 'PRIMARY KEY'
AND t.table_schema = DATABASE() AND t.table_name = 'cats';
4）建表
SHOW CREATE TABLE dba.cats;
DESCRIBE cats;
5) 增加表同步列
ALTER TABLE `cats` ADD COLUMN `sys_sync_id` INT AUTO_INCREMENT UNIQUE;
ALTER TABLE `cats` ADD COLUMN `sys_sync_update_time`  TIMESTAMP ON UPDATE CURRENT_TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE `cats` ADD COLUMN `sys_sync_create_time`  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
 */

func main() {
    dbReplicConfig := readDbReplicConfig()
    dbFrom := mydb.GetDb(dbReplicConfig.DbFrom)
    defer dbFrom.Close()
    dbTo := mydb.GetDb(dbReplicConfig.DbTo)
    defer dbTo.Close()

    nodb, tempDir, _ := mynodb.OpenTemp()
    defer os.RemoveAll(tempDir)

    gocron.Every(5).Seconds().Do(mainTask, dbReplicConfig, nodb, dbFrom, dbTo)
    <-gocron.Start()
}

func mainTask(dbReplicConfig DbReplicConfig, nodb *mynodb.Nodb, dbFrom, dbTo *mydb.Db) {
    fmt.Println(dbReplicConfig)
}

type DbReplicConfig struct {
    DbFrom        string `toml:"dbFrom"`
    DbTo          string `toml:"dbTo"`
    ExcludeTables []string `toml:"excludeTables"`
}

func readDbReplicConfig() DbReplicConfig {
    fpath := "dbreplic.toml"
    if len(os.Args) > 1 {
        fpath = os.Args[1]
    }

    dbReplicConfig := DbReplicConfig{}
    if _, err := toml.DecodeFile(fpath, &dbReplicConfig); err != nil {
        myutil.CheckErr(err)
    }

    return dbReplicConfig
}
