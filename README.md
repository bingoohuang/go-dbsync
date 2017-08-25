# go-dbsync
synchronize the database tables

## V1
编译:

1. 获取MySQL驱动 `go get github.com/go-sql-driver/mysql`
2. 本地编译 `go build src/user-sync.go` 
3. 本地编译Linux版本(bash) `GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o user-sync.linux src/user-sync.go`

只能同步tr_f_user表的数据。
同步原理：比较两边数据，没有的补充上，有的但是不一样的，控制台打印。
用法:

`./user-sync "root:my-secret-pw@tcp(192.168.99.100:13306)/dba" "root:my-secret-pw@tcp(192.168.99.100:13306)/dbb"`

## V2

1. 获取MySQL驱动 `go get github.com/go-sql-driver/mysql`
2. 获取nodb `go get github.com/lunny/nodb`
3. 获取toml `go get github.com/BurntSushi/toml`
4. 本地编译 `go build src/dbsync.go` 
5. 本地编译Linux版本(bash) `GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o dbsync.linux src/dbsync.go`


可以同步指定表的数据,例如:

```
~/g/go-dbsync > ./dbsync config/dbsync.toml
<<<map[openid:NULL merchant_id:0 create_time:2016-05-16 10:34:22 effective:1 user_id:a045719884460976128 mobile:18231869455]
>>>map[create_time:2016-05-16 10:34:22 effective:1 user_id:a045719884460976128 mobile:18231869455 openid: merchant_id:0]
<<<map[create_time:2016-08-04 14:36:36 effective:1 user_id:b217018113249683456 mobile:18231684453 openid: merchant_id:18D72F678EFA]
>>>map[user_id:b217018113249683456 mobile:18231684453 openid:NULL merchant_id:18D72F678EFA create_time:2016-08-04 14:36:36 effective:1]
Start to merge left tr_f_user, merged 30 rows
Start to merge right tr_f_user, merged 15 rows
```

### 配置文件格式

```toml
# 同步的两个库配置
Db1 = "root:my-secret-pw@tcp(192.168.99.100:13306)/dba"
Db2 = "root:my-secret-pw@tcp(192.168.99.100:13306)/dbb"
# 需要同步的表,可以指定多个
SyncTables = [ "tr_f_user" ]
```

# go-blackcat-web
提供了一个blackcat的消息跟踪展示原始的web<br>
编译: `env GOOS=linux GOARCH=amd64 go build -o go-blackcat-web-linux.bin src/go-blackcat-web.go` <br>
配置文件go-blackcat-web.toml:

```toml
CassandraHosts = ["127.0.0.1"]
CassandraPort = 9042
ListenPort = 8181
```
启动: `./nohup ./go-blackcat-web-linux ./go-blackcat-web.toml 2>&1 > go-blackcat-web.log  &`


# go-ip-allow
build:`env GOOS=linux GOARCH=amd64 go build -o go-ip-allow.linux.bin src/go-ip-allow.go`<br/>
config file go-ip-allow.toml:

```toml
Envs = [ "DEV", "TEST", "DEMO", "PRODUCT" ]
Mobiles = ["18212345678"]
MobileTags = ["BINGOO"]
ListenPort = 8182
SendCaptcha = "http://127.0.0.1:8020/v1/notify/send-captcha"
VerifyCaptcha = "http://127.0.0.1:8020/v1/notify/verify-captcha"
UpdateFirewallShell = "/home/ci/firewall/iphelp.sh"
```
bash scripts:
```bash
export http_proxy=http://127.0.0.1:9999
export https_proxy=http://127.0.0.1:9999
go get -v -u github.com/BurntSushi/toml
go get -v -u gopkg.in/kataras/iris.v6
```
fish scripts:
```fish
set -x http_proxy http://127.0.0.1:9999
set -x https_proxy http://127.0.0.1:9999
go get -v -u github.com/BurntSushi/toml
go get -v -u gopkg.in/kataras/iris.v6
```

# go-tail-web
build:<p>`env GOOS=linux GOARCH=amd64 go build -o go-tail-web.linux.bin src/tailweb/go-tail-web.go`</p>
run:<p>`nohup ./go-tail-web.linux.bin -log=/Users/bingoo/gitlab/et-server/et.log -port=8497 > go-tail-web.out 2>&1 &`</p>
or multiple logs:<p>`nohup ./go-tail-web.linux.bin -log=/Users/bingoo/gitlab/et-server/et.log,/Users/bingoo/gitlab/ab.log -port=8497 > go-tail-web.out 2>&1 &`</p>
or multiple logs with log naming:<p>`nohup ./go-tail-web.linux.bin -log=et:/Users/bingoo/gitlab/et-server/et.log,ab:/Users/bingoo/gitlab/ab.log -port=8497 > go-tail-web.out 2>&1 &`</p>

# go log collector
## go-log-client
build:<p>`env GOOS=linux GOARCH=amd64 go build -o go-log-client.linux.bin src/logclient/go-log-client.go`</p>
run:<p>`nohup ./go-log-client.linux.bin -server=127.0.0.1 -port=10811 -log=et:/Users/bingoo/gitlab/et-server/et.log,ab:/Users/bingoo/gitlab/ab.log > go-log-client.out 2>&1 &`</p>
## go-log-server
build:<p>`env GOOS=linux GOARCH=amd64 go build -o go-log-server.linux.bin src/logserver/go-log-server.go`</p>
run:<p>`nohup ./go-log-server.linux.bin  -port=10811 > go-log-server.out 2>&1 &`</p>
All the logs collected from go-log-client will append to related log files with specified naming, like et.log, ab.log and etc.

![image](https://user-images.githubusercontent.com/1940588/28238816-9745199c-698e-11e7-8ed5-f925130a0826.png)

# go sql web
1. install go-bindata: `go get -u -v github.com/jteeuwen/go-bindata/...`
2. install goimports: `go get -u -v golang.org/x/tools/cmd/goimports`
3. build: `env GOOS=linux GOARCH=amd64 go build -o go-sql-web.linux.bin`
4. run: `./sqlweb -dataSource="user:pass@tcp(ip:3306)/db?charset=utf8" -cookieName=customizedCookieName -key=size16encryptKey -corpId=wx_corpId -corpSecret=wx_secret -agentId=wx_agentId -redirectUri=redirectUri`

# go redis web
try to implement redis web in go like [phpRedisAdmin](https://github.com/erikdubbelboer/phpRedisAdmin).

