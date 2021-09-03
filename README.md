# econvert
MySQL Storage engine conversion

## **简介**

​		此工具用于MySQL存储引擎转换，支持CTAS和ALTER两种模式，目前只支持MyISAM和InnoDB存储引擎相互转换，其它引擎尚不支持。



​		注意：当对表进行引擎转换时，建议业务停止访问或者极少量访问时进行。

## **原理**

​		CTAS模式会创建一个新表，然后把业务表数据insert到新表中。

​		ALTER是直接对业务表执行alter字句来进行转换。




## **GO版本要求**

​		GO版本要求在1.10以上



## 依赖

> ```go
> module econvert
> 
> go 1.15
> 
> require (
> 	github.com/eiannone/keyboard v0.0.0-20200508000154-caf4b762e807 // indirect
> 	github.com/go-sql-driver/mysql v1.6.0
> 	github.com/nsf/termbox-go v1.1.1 // indirect
> 	golang.org/x/sys v0.0.0-20210902050250-f475640dd07b // indirect
> )
> ```
>

​		注意：构建时应打开go mod模式(set GO111MODULE=on)，打开后将自动联网下载相关依赖包



## 编译

1. #### 构建

> ```shell
> $ go env -w GO111MODULE=on
> $ go env -w GOPROXY=https://goproxy.io,direct
> # 因为国内的网络限制，需要使用代理拉取相关依赖代码。
> 
> # ls -l
> -rw-r--r--. 1 root root 21924 Sep  3 11:11 econvert.go
> -rw-r--r--. 1 root root   265 Sep  3 11:11 go.mod
> -rw-r--r--. 1 root root  1606 Sep  3 11:11 go.sum
> -rw-r--r--. 1 root root   536 Sep  3 11:11 makefile
> -rw-r--r--. 1 root root 14511 Sep  3 11:11 README.md
> 
> # make
> go build -gcflags=all='-l -N' -ldflags " \
> 	-X 'main.Platform=linux amd64' \
> 	-X 'main.BuildTime=Fri Sep  3 11:14:13 EDT 2021' \
> 	-X 'main.GoVersion=go version go1.14 linux/amd64' \
> 	-X 'main.VERSION=1.02'" -o ../build/econvert/bin/econvert
> ```
>

​		提示：环境及版本信息会写入到二进制文件，编译完成后不可更改



2. #### 二进制文件

​		编译完成后将在此目录中生成二进制执行文件。

> ```shell
> # ls -l ../build/econvert/bin/econvert 
> -rwxr-xr-x. 1 root root 6673893 Sep  3 11:14 ../build/econvert/bin/econvert
> 
> # ldd ../build/econvert/bin/econvert 
> 	linux-vdso.so.1 =>  (0x00007fff23dff000)
> 	libpthread.so.0 => /lib64/libpthread.so.0 (0x0000003a3a600000)
> 	libc.so.6 => /lib64/libc.so.6 (0x0000003a3a200000)
> 	/lib64/ld-linux-x86-64.so.2 (0x0000003a39a00000)
> ```
>

​		注意：此执行文件不可以进行任何编辑，否则将发生损坏导致无法运行。



## 示例

1. #### 转换ucds下所有表从MyISAM引擎到InnoDB引擎

```shell
#econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert yes
```



2. #### 转换ucds下所有表从MyISAM引擎到InnoDB引擎，并删除由于转换引擎时所产生的临时表，默认不删除。

```shell
#econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert yes \
-clean yes
```



3. #### 只对某一个表进行引擎转换

```shell
./econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert no \
-clean yes \
-table test01
```



4. #### 基于全库排除某一个表

```shell
./econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert no \
-clean yes \
-exclude test02
```



5. #### 选择转换引擎模式，默认是CTAS模式，CTAS模式会产生一个临时表，并且当业务进行时可能导致数据丢失。当使用CTAS模式时，默认情况下当表小于300M时强制使用ALTER模式来进行引擎转换。当配置-size 0 时，将会禁用强制alter模式，那么此时所有表都将采用CTAS模式来转换。-size参数只有当CTAS模式才有效，alter模式是无效的。

```shell
./econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert no \
-clean yes \
-exclude test02
-size 0
```



6. #### 转换引擎失败后继续，默认情况下一旦发生失败，所有转换操作立即终止，例如期望当发生3次失败后终止，可使用如下参数。

```shell
./econvert -host 172.168.120.112 \
-user root \
-password 'root' \
-cdb ucds \
-fromengine myisam \
-toengine innodb \
-convert no \
-clean yes \
-method alter \
-errcount 3
```



7. #### 终端：重新加载表

```shell
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
| SEQ | TABLE_SCHEMA |               TABLE_NAME | TABLE_TYPE | ENGINE | SIZE_MB | METHOD |    STATE |  WAIT_TIME |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
|   1 |         ucds |          lender_set_1001 | BASE TABLE | MyISAM |    1209 |        |    Ready |          0 |
|   2 |         ucds | ent_record_callback_info | BASE TABLE | MyISAM |     242 |        |    Ready |          0 |
|   3 |         ucds |              ipvs_tb1010 | BASE TABLE | MyISAM |      37 |        |    Ready |          0 |
|   4 |         ucds |              ipvs_tb1011 | BASE TABLE | MyISAM |      24 |        |    Ready |          0 |
|   5 |         ucds |                   test06 | BASE TABLE | MyISAM |      19 |        |    Ready |          0 |
|   6 |         ucds |                   test02 | BASE TABLE | MyISAM |      17 |        |    Ready |          0 |
|   7 |         ucds |                  test991 | BASE TABLE | MyISAM |      17 |        |    Ready |          0 |
|   8 |         ucds |                      bak | BASE TABLE | MyISAM |      11 |        |    Ready |          0 |
|   9 |         ucds |                   test10 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  10 |         ucds |                   test09 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  11 |         ucds |                   test08 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  12 |         ucds |                   test13 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  13 |         ucds |                   test12 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+

The available command sets are as follows
 y    Perform storage engine conversion
 e    Exit storage engine conversion
 p    Re output engine conversion table
 c    Re output database configuration information
 r    Reload transformation engine table
 h    Output command information
 			------->> 如果此时test01表被删除了，我们可以在不退出进程的情况下输入r指令来重载表。
QNCLI> r
Table reloading is completed. A total of 12 tables are loaded


QNCLI> p	------->> 重载完成后可以输入p指令查看表集，此时将会发生test01表已经消失
Database table information statistics:
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
| SEQ | TABLE_SCHEMA |               TABLE_NAME | TABLE_TYPE | ENGINE | SIZE_MB | METHOD |    STATE |  WAIT_TIME |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
|   1 |         ucds |          lender_set_1001 | BASE TABLE | MyISAM |    1209 |        |    Ready |          0 |
|   2 |         ucds | ent_record_callback_info | BASE TABLE | MyISAM |     242 |        |    Ready |          0 |
|   3 |         ucds |              ipvs_tb1010 | BASE TABLE | MyISAM |      37 |        |    Ready |          0 |
|   4 |         ucds |              ipvs_tb1011 | BASE TABLE | MyISAM |      24 |        |    Ready |          0 |
|   5 |         ucds |                   test06 | BASE TABLE | MyISAM |      19 |        |    Ready |          0 |
|   6 |         ucds |                   test02 | BASE TABLE | MyISAM |      17 |        |    Ready |          0 |
|   7 |         ucds |                      bak | BASE TABLE | MyISAM |      11 |        |    Ready |          0 |
|   8 |         ucds |                   test13 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|   9 |         ucds |                   test12 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  10 |         ucds |                   test10 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  11 |         ucds |                   test09 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  12 |         ucds |                   test08 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+

QNCLI> 

			------->> 如果此时创建了一个新表，同样也可以输入r键来重载。
QNCLI> r
Table reloading is completed. A total of 13 tables are loaded

QNCLI> p
Database table information statistics:
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
| SEQ | TABLE_SCHEMA |               TABLE_NAME | TABLE_TYPE | ENGINE | SIZE_MB | METHOD |    STATE |  WAIT_TIME |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
|   1 |         ucds |          lender_set_1001 | BASE TABLE | MyISAM |    1209 |        |    Ready |          0 |
|   2 |         ucds | ent_record_callback_info | BASE TABLE | MyISAM |     242 |        |    Ready |          0 |
|   3 |         ucds |              ipvs_tb1010 | BASE TABLE | MyISAM |      37 |        |    Ready |          0 |
|   4 |         ucds |              ipvs_tb1011 | BASE TABLE | MyISAM |      24 |        |    Ready |          0 |
|   5 |         ucds |                   test06 | BASE TABLE | MyISAM |      19 |        |    Ready |          0 |
|   6 |         ucds |                   test02 | BASE TABLE | MyISAM |      17 |        |    Ready |          0 |
|   7 |         ucds |                  test991 | BASE TABLE | MyISAM |      17 |        |    Ready |          0 |
|   8 |         ucds |                      bak | BASE TABLE | MyISAM |      11 |        |    Ready |          0 |
|   9 |         ucds |                   test10 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  10 |         ucds |                   test09 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  11 |         ucds |                   test08 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  12 |         ucds |                   test13 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
|  13 |         ucds |                   test12 | BASE TABLE | MyISAM |       2 |        |    Ready |          0 |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+

QNCLI> 	
```



8. #### 终端：其它指令

```sql
QNCLI> h			------->> h指令输出帮助信息
The available command sets are as follows
 y    Perform storage engine conversion
 e    Exit storage engine conversion
 p    Re output engine conversion table
 c    Re output database configuration information
 r    Reload transformation engine table
 h    Output command information

QNCLI> c			------->> c指令重新输出数据库配置信息
Database Connection Authentication Information:
      HOST: 172.168.120.112
      Port: 3306
      User: root
  PASSWORD: root
       CDB: ucds
FROMENGINE: myisam
  TOENGINE: innodb
   CONVERT: yes
     TABLE: 
   EXCLUDE: 
    METHOD: alter
ERRORCOUNT: 3
     CLEAN: yes
      SIZE: 300 (In CTAS mode, the table is less than 300MB, and the ALTER mode is forced)

QNCLI> y			------->> y指令执行引擎转换
SQL> alter table `ucds`.`lender_set_1001` engine=innodb;
Table altered. 6485000 rows affected (498439 Millisecond)

SQL> alter table `ucds`.`ent_record_borror_info` engine=innodb;
Table altered. 1297000 rows affected (208690 Millisecond)

SQL> alter table `ucds`.`ipvs_tb1010` engine=innodb;
Table altered. 1933312 rows affected (27352 Millisecond)

SQL> alter table `ucds`.`ipvs_tb1011` engine=innodb;
Table altered. 1933312 rows affected (52270 Millisecond)

SQL> alter table `ucds`.`test06` engine=innodb;
Table altered. 165888 rows affected (28560 Millisecond)

SQL> alter table `ucds`.`test02` engine=innodb;
Table altered. 147456 rows affected (20803 Millisecond)

SQL> alter table `ucds`.`test991` engine=innodb;
Table altered. 147456 rows affected (21911 Millisecond)

SQL> alter table `ucds`.`bak` engine=innodb;
Table altered. 92160 rows affected (18722 Millisecond)

SQL> alter table `ucds`.`test10` engine=innodb;
Table altered. 18432 rows affected (14028 Millisecond)

SQL> alter table `ucds`.`test09` engine=innodb;
Table altered. 18432 rows affected (2724 Millisecond)

SQL> alter table `ucds`.`test08` engine=innodb;
Table altered. 18432 rows affected (11810 Millisecond)

SQL> alter table `ucds`.`test13` engine=innodb;
Table altered. 18432 rows affected (2462 Millisecond)

SQL> alter table `ucds`.`test12` engine=innodb;
Table altered. 18432 rows affected (2616 Millisecond)

Database table information statistics:
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
| SEQ | TABLE_SCHEMA |               TABLE_NAME | TABLE_TYPE | ENGINE | SIZE_MB | METHOD |    STATE |  WAIT_TIME |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
|   1 |         ucds |          lender_set_1001 | BASE TABLE | MyISAM |    1209 |  ALTER |  SUCCESS |     498439 |
|   2 |         ucds | ent_record_callback_info | BASE TABLE | MyISAM |     242 |  ALTER |  SUCCESS |     208690 |
|   3 |         ucds |              ipvs_tb1010 | BASE TABLE | MyISAM |      37 |  ALTER |  SUCCESS |      27352 |
|   4 |         ucds |              ipvs_tb1011 | BASE TABLE | MyISAM |      24 |  ALTER |  SUCCESS |      52270 |
|   5 |         ucds |                   test06 | BASE TABLE | MyISAM |      19 |  ALTER |  SUCCESS |      28560 |
|   6 |         ucds |                   test02 | BASE TABLE | MyISAM |      17 |  ALTER |  SUCCESS |      20803 |
|   7 |         ucds |                  test991 | BASE TABLE | MyISAM |      17 |  ALTER |  SUCCESS |      21911 |
|   8 |         ucds |                      bak | BASE TABLE | MyISAM |      11 |  ALTER |  SUCCESS |      18722 |
|   9 |         ucds |                   test10 | BASE TABLE | MyISAM |       2 |  ALTER |  SUCCESS |      14028 |
|  10 |         ucds |                   test09 | BASE TABLE | MyISAM |       2 |  ALTER |  SUCCESS |       2724 |
|  11 |         ucds |                   test08 | BASE TABLE | MyISAM |       2 |  ALTER |  SUCCESS |      11810 |
|  12 |         ucds |                   test13 | BASE TABLE | MyISAM |       2 |  ALTER |  SUCCESS |       2462 |
|  13 |         ucds |                   test12 | BASE TABLE | MyISAM |       2 |  ALTER |  SUCCESS |       2616 |
+-----+--------------+--------------------------+------------+--------+---------+--------+----------+------------+
```



## 参数说明
1. #### 参数帮助信息


> ```shell
> Usage of ../build/econvert/bin/econvert:
>   -cdb string
>     	Specify the DB to be converted
>   -clean string
>     	Clear temporary table (default "no")
>   -convert string
>     	Perform engine conversion? (default "no")
>   -errcount string
>     	Stop running after reaching the number of times, 0 disabled (default "0")
>   -exclude string
>     	Specify excluded tables, multiple tables are separated by commas
>   -fromengine string
>     	Specify conversion source engine (default "myisam")
>   -host string
>     	Specify the database host address (default "localhost")
>   -method string
>     	Select engine conversion method. Available values: CTAS, ALTER (default "CTAS")
>   -password string
>     	Specify database authentication password
>   -port int
>     	Specify the database host port (default 3306)
>   -size string
>     	Force alter when the table size is MB?. Only when method is CTAS (default "300")
>   -table string
>     	Specify the tables to be converted. Multiple tables are separated by commas
>   -toengine string
>     	Specify conversion target engine (default "innodb")
>   -user string
>     	Specify database authentication user (default "root")
> ```



2. #### 终端指令说明

> ```shell
>  y    Perform storage engine conversion
>  e    Exit storage engine conversion
>  p    Re output engine conversion table
>  c    Re output database configuration information
>  r    Reload transformation engine table
>  h    Output command information
> ```
