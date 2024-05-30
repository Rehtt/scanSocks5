# scan socks5
一个简单的socks5扫描程序

## 安装

### go install
```shell
go install github.com/Rehtt/scanSocks5@latest
```
or
```shell
git clone github.com/Rehtt/scanSocks5
cd scanSocks5
go install
```

## 使用
可配合zmap使用对全网进行socks5端口扫描

### 使用文件的方式
```shell
# 使用zmap随机扫描10000000个ip的7890端口，将结果ip输出到ip.txt
zmap -n 10000000 -p 7890 -o ip.txt
# 使用scanSocks5对ip.txt文件中的ip进行[7890 7891 7892]端口扫描，开启20协程，过滤扫描美国的ip
scanSocks5 -i ./ip.txt -limit 20 -region 美国 -ports 7890,7891,7892
```

### 使用linux管道的方式
```shell
zmap -n 10000000 -p 7890 -q -o - | scanSocks5 -i - -limit 20 -region 美国 -ports 7890,7891,7892
```
