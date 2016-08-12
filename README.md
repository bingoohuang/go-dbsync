# go-dbsync
synchronize the database tables

## V1
编译:

1. 获取MySQL驱动 `go get github.com/go-sql-driver/mysql`
2. 本地编译 `go build src/user-sync.go` 
3. Linux 编译 `GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o user-sync.linux src/user-sync.go`

只能同步tr_f_user表的数据。
同步原理：比较两边数据，没有的补充上，有的但是不一样的，控制台打印。
用法:

`./user-sync "root:my-secret-pw@tcp(192.168.99.100:13306)/dba" "root:my-secret-pw@tcp(192.168.99.100:13306)/dbb"`

## V2

可以同步指定表的数据,例如:

```
~/g/go-dbsync > ./dbsync "root:my-secret-pw@tcp(192.168.99.100:13306)/dba" "root:my-secret-pw@tcp(192.168.99.100:13306)/dbb" tr_f_user
<<<map[openid:NULL merchant_id:0 create_time:2016-05-16 10:34:22 effective:1 user_id:a045719884460976128 mobile:18231869455]
>>>map[create_time:2016-05-16 10:34:22 effective:1 user_id:a045719884460976128 mobile:18231869455 openid: merchant_id:0]
<<<map[create_time:2016-08-04 14:36:36 effective:1 user_id:b217018113249683456 mobile:18231684453 openid: merchant_id:18D72F678EFA]
>>>map[user_id:b217018113249683456 mobile:18231684453 openid:NULL merchant_id:18D72F678EFA create_time:2016-08-04 14:36:36 effective:1]
Start to merge left tr_f_user, merged 30 rows
Start to merge right tr_f_user, merged 15 rows
```
