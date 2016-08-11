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
