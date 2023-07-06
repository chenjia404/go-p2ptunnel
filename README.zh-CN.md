[English](./README.md) | [简体中文](./README.zh-CN.md) 
## go-p2ptunnel

使用libp2p建立tcp隧道(不支持udp)，底层传输可以使用quic、tcp、websocket、webtransport实现，使用 noise 协议加密传输，自带nat，可以多层组合使用。

如果在没有公网ip的情况下，可以使用节点id进行连接，如果有情况的直接指定ip和通讯协议进行连接。

节点私钥文件在当前目录下，文件名后缀是 .key ，默认是 user.key ，删除后重启程序就会自动创建新的节点id。

### 参数说明

| 字段        | 类型          | 说明                                                                     |
|-----------|-------------|------------------------------------------------------------------------|
| l         | ip          | 监听或者转发的地址，如果没有设置id，就是转发这个地址，如果设置了，就是把远程端口映射到本地这个地址，默认值 127.0.0.1:10086 |
| id        | multiaddr   | 连接远程服务id                                                               |
| p2p_port  | ip          | p2p使用的端口，也是监听其它节点连接的端口，默认4001，会自动进行nat，但是可能需要您进行端口映射。如果是0，就会选择随机端口     |
| nodisc    | bool        | 禁止广播提高性能，连接节点必须使用带ip和端口的链接                                             |
| user      | string      | 指定使用本地的哪一个key文件                                                        |
| update    | bool        | 从GitHub更新最新版，会验证升级包签名、sha512                                           |
| auto_update    | bool        | 开启自动更新                                           |
| max_peers | int         | 最大连接数，默认500                                                            |
| socks5 | ip         | socks5监听ip，例如 127.0.0.1:10086，如果l字段为空，就使用这个字段                          |

### 流量特征

如果没有关闭节点广播(-nodisc)，节点会和大量节点进行通信，就如同一个普通的p2p程序，一天时间可能会和几千个ip通信，但是每个ip使用的流量在几十kb到几百kb左右。

### 使用案例

如果你的的公司或者学校网络限制了一些网站使用，那么你搭建一个隧道，连接到服务器就可以无限制的使用了。在公司网管来看，你只是使用了一个普通的p2p程序，而且连接了多个ip。

和服务器连接的过程，你可以使用quic、tcp、websocket、webtransport这几种协议的任意一种，根据你的网络情况来选择。在连接id构建上，如果只是服务器节点id，就不固定使用的网络连接方式，如果用 tcp 的连接地址就会使用 tcp 链接，也可以用 websocket 格式的连接地址。

例如你在服务器上有一个服务，监听的127.0.0.1:38080，在服务器`./go-p2ptunnel -l 127.0.0.1:38080 -p2p_port 4001 -nodisc`，然后复制输出地址，选择其中的一个，例如 `/ip4/1.2.3.4/tcp/4002/p2p/12D3KooWJTa5peaDcNHLuzSXLt6VQ9JFyWVG5hM2NVJZjBQTUhd5`，当然你也可以直接 12D3KooWJTa5peaDcNHLuzSXLt6VQ9JFyWVG5hM2NVJZjBQTUhd5 。

本地执行 `./go-p2ptunnel -id 12D3KooWJTa5peaDcNHLuzSXLt6VQ9JFyWVG5hM2NVJZjBQTUhd5 -l 127.0.0.1:10089`，然后你本地连接这个端口就可以使用这个服务器。

服务器的服务可以是一个数据库，也可以是一个后台，只要是tcp协议的就可以。

### 编译

` go build -trimpath -ldflags="-w -s" `

### 升级

`./go-p2ptunnel -update`

v0.0.6 以后，程序会自动从GitHub更新最新版版本，会验证文件的sha512和gpg签名，gpg签名id为 `189BE79683369DA3`

### id格式(multiaddr)
|  类型 | 样例|说明  |
|  ----  | ----  |----  |
|12D3KooWLHjy7D    | 纯id| 只知道id，不知道协议、ip这些 |
|/p2p/12D3KooWLHjy7D|纯id | 只知道id，不知道协议、ip这些|
|/ip4/1.1.1.1/tcp/4001/p2p/12D3KooWLHjy7D| 详细路径|知道ip、协议，使用的tcp |
|/ip4/1.1.1.1/udp/4001/quic-v1/p2p/12D3KooWLHjy7D| 详细路径|知道ip、协议，使用的quic |

节点启动的时候会输出相应的地址，把里面的 ip 修改成公网ip即可。

可以通过路径里面的tcp、quic控制连接行为。

### 打开本地端口
`./go-p2ptunnel -l 127.0.0.1:3389`

注意这里会输出你的节点id，然后通过聊天软件发给你的朋友，这里假设id是12D3。

这里需要完整的ip加端口。

### 连接
`./go-p2ptunnel -id 12D3 -l 127.0.0.1:10089`

连接可能需要几秒到1分钟，连接成功后，就把远程端口映射到了 127.0.0.1:10089 

然后朋友在远程桌面连接 127.0.0.1:10089 即可。

### 打包

`goreleaser release --skip-publish --skip-validate --clean`

### 验证签名

```
gpg --recv-key E1346252ED662364CA37F716189BE79683369DA3

gpg --verify .\ethtweet_0.7.4_windows_amd64.zip.asc .\ethtweet_0.7.4_windows_amd64.zip
```
如果出现`using RSA key E1346252ED662364CA37F716189BE79683369DA3`就是验证成功

### 服务发布和分享(todo)

可以把你的服务发布出去，其它节点搜索服务名后，连接进行使用。必须是基于tcp的服务，暂不支持udp。

服务命名 `本应用的Protocol + / + 服务名的Protocol`,如果不是标准知名协议，建议使用类似包名的形式，避免服务冲突。



## 注意事项

1.本应用虽然使用的端对端加密，但是不保证传输数据的安全性，重要数据请勿使用本应用传递。

2.由于是p2p隧道，所以本程序会连接多个ip，如果介意，请使用frp。

3.如果有多个客户端连接，请加大服务端的最大文件数，不然可能导致连接数不够。

## 上游项目

[go-libp2p](https://github.com/libp2p/go-libp2p)