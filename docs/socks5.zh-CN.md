## 使用go-p2ptunnel搭建一个socks5的代理

首先在服务器上下载 [https://github.com/chenjia404/go-p2ptunnel/releases](https://github.com/chenjia404/go-p2ptunnel/releases)

创建单独的目录

mkdir go-p2ptunnel

下载最新版并解压

```
curl -L $( curl -L https://api.github.com/repos/chenjia404/go-p2ptunnel/releases/latest |grep browser_ |grep "go-p2ptunnel" |grep -i $(uname -s)|grep -v asc |sed 's/amd/x86_/' |grep $(uname -m) | sed 's/x86_/amd/' |cut -d\" -f4)   -o go-p2ptunnel.tar.gz

tar -xvzf go-p2ptunnel.tar.gz`
```
签名验证（可选）
```
curl -L $( curl -L https://api.github.com/repos/chenjia404/go-p2ptunnel/releases/latest |grep browser_ |grep "go-p2ptunnel" |grep -i $(uname -s)|grep  asc |sed 's/amd/x86_/' |grep $(uname -m) | sed 's/x86_/amd/' |cut -d\" -f4)   -o go-p2ptunnel.tar.gz.asc

gpg --verify go-p2ptunnel.tar.gz.asc  go-p2ptunnel.tar.gz
```
如果出现`using RSA key E1346252ED662364CA37F716189BE79683369DA3`就是验证成功

启动程序`./go-p2ptunnel -nodisc -auto_update -socks5 127.0.0.1:10086`

你会看到类似这样的输出

```
p2ptunnel 0.2.21-"70ab671"
buildTime "2023-07-07T07:23:58Z"
System version: amd64/windows
Golang version: go1.20.5
multiaddr:/ip4/127.0.0.1/tcp/4001
multiaddr:/ip4/127.0.0.1/tcp/4002/ws
multiaddr:/ip4/192.168.1.33/tcp/18016
multiaddr:/ip4/192.168.1.33/tcp/18683/ws
multiaddr:/ip4/192.168.31.248/tcp/4001
multiaddr:/ip4/192.168.31.248/tcp/4002/ws
multiaddr:/ip6/::1/tcp/4001
multiaddr:/ip6/::1/tcp/4002/ws
multiaddr:/ip6/2001:0:2851:fcb0:2852:937:8385:ae17/tcp/4001
multiaddr:/ip6/2001:0:2851:fcb0:2852:937:8385:ae17/tcp/4002/ws
Your id: 12D3KooWQEAhRn5TnsUxKEnXwqZKaNNRNk9v8qaQeRk4LRtMyQVi
```

其中 `/ip4/192.168.31.248/tcp/4001` 里面的ip修改成服务器ip，加上Your id后面的内容，拼接成 `/ip4/192.168.31.248/tcp/4001/p2p/12D3KooWQEAhRn5TnsUxKEnXwqZKaNNRNk9v8qaQeRk4LRtMyQVi`。

本地客户端启动：

`./go-p2ptunnel -id /ip4/192.168.31.248/tcp/4002/p2p/12D3KooWQEAhRn5TnsUxKEnXwqZKaNNRNk9v8qaQeRk4LRtMyQVi -l 0.0.0.0:10080 -auto_update`
然后在需要代理的程序里面设置socks5代理为127.0.0.1:10080
然后客户端输出如下：

```
2023-07-10 00:58:04 2023/07/09 16:58:04 Stream:8
2023-07-10 00:58:04 2023/07/09 16:58:04 open New Stream
2023-07-10 00:58:04 2023/07/09 16:58:04 New Stream is open
2023-07-10 00:58:05 2023/07/09 16:58:05 Stream:9
2023-07-10 00:58:05 2023/07/09 16:58:05 open New Stream
2023-07-10 00:58:05 2023/07/09 16:58:05 New Stream is open
2023-07-10 00:58:05 新请求
2023-07-10 00:58:05 新请求
2023-07-10 00:58:05 2023/07/09 16:58:05 Stream:9
2023-07-10 00:58:05 2023/07/09 16:58:05 open New Stream
2023-07-10 00:58:05 2023/07/09 16:58:05 New Stream is open
2023-07-10 00:58:06 新请求
```

这时候你已经成功的使用本程序搭建代理了，本程序有连接复用，一个客户端只建立一个链接，然后多个流，避免部分程序建立大量连接导致网络中断。
另外整个连接过程都是加密的，前面使用的id就类似密码。