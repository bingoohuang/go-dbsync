# go-dbsync
synchronize the database tables

## V1
只能同步tr_f_user表的数据。
同步原理：比较两边数据，没有的补充上，有的但是不一样的，控制台打印。
用法:

`./user-sync "root:my-secret-pw@tcp(192.168.99.100:13306)/dba" "root:my-secret-pw@tcp(192.168.99.100:13306)/dbb"`
