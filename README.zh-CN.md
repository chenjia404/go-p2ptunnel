[English](./README.md) | [简体中文](./README.zh-CN.md) 
## go-p2ptunnel

使用libp2p建立tcp隧道(不支持udp)，底层传输可以使用quic、tcp、package websocket、webtransport实现，使用 noise 协议加密传输，自带nat，可以多层组合使用。

如果在没有公网ip的情况下，可以使用节点id进行连接，如果有情况的直接指定ip和通讯协议进行连接。

节点私钥文件在当前目录下，文件名后缀是 .key ，默认是 user.key ，删除后重启程序就会自动创建新的节点id。

### 参数说明

| 字段      | 类型           | 说明                                                                     |
|---------|--------------|------------------------------------------------------------------------|
| l       | 地址           | 监听或者转发的地址，如果没有设置id，就是转发这个地址，如果设置了，就是把远程端口映射到本地这个地址，默认值 127.0.0.1:10086 |
| id      | multiaddr格式的 | 连接远程服务id                                                               |
| p2p_port | ip端口         | p2p使用的端口，也是监听其它节点连接的端口，默认4001，会自动进行nat，但是可能需要您进行端口映射                   |
| nodisc  | bool         | 禁止广播提高性能，连接节点必须使用带ip和端口的链接                                             |
| user    | 字符串          | 指定使用本地的哪一个key文件                                                        |
| update  | bool          | 从GitHub更新最新版，会验证升级包签名、sha512                                           |

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

`goreleaser release --skip-publish  --rm-dist`


### 服务发布和分享(todo)

可以把你的服务发布出去，其它节点搜索服务名后，连接进行使用。必须是基于tcp的服务，暂不支持udp。

服务命名 `本应用的Protocol + / + 服务名的Protocol`,如果不是标准知名协议，建议使用类似包名的形式，避免服务冲突。



## 注意事项

1.本应用虽然使用的端对端加密，但是不保证传输数据的安全性，重要数据请勿使用本应用传递。

2.由于是p2p隧道，所以本程序会连接多个ip，如果介意，请使用frp。

3.如果有多个客户端连接，请加大服务端的最大文件数，不然可能导致连接数不够。

## 上游项目

[go-libp2p](https://github.com/libp2p/go-libp2p)