[English](./README.md) | [简体中文](./README.zh-CN.md)

## go-p2ptunnel

Use libp2p to establish a tcp tunnel (does not support udp), the underlying transmission can be realized using quic, tcp, package websocket, webtransport, use the noise protocol to encrypt the transmission, comes with nat, and can be used in multi-layer combinations.

If there is no public network ip, you can use the node id to connect. If there is a situation, directly specify the ip and communication protocol to connect.

The node private key file is in the current directory, the suffix of the file name is .key , the default is user.key , and restarting the program after deletion will automatically create a new node id.

### Parameter Description

| Field | Type | Description |
|---------|-------------|---------------------------------------------------------------------------|
| l | Address | The listening or forwarding address. If the id is not set, the address is forwarded. If it is set, the remote port is mapped to the local address. The default value is 127.0.0.1:10086 |
| id | multiaddr format | connection remote service id |
| p2p_port | ip port | The port used by p2p is also the port that listens for other nodes to connect. The default is 4001, and it will automatically perform nat, but you may need to perform port mapping |
| nodisc | bool | Prohibit broadcasting to improve performance, connecting nodes must use links with ip and port |
| user | string | specify which local key file to use |
| update | bool | Update the latest version from GitHub, it will verify the upgrade package signature, sha512 |

### build

` go build -trimpath -ldflags="-w -s" `

### upgrade

`./go-p2ptunnel -update`

After v0.0.6, the program will automatically update the latest version from GitHub, and verify the sha512 and gpg signature of the file. The gpg signature id is `189BE79683369DA3`

### id format(multiaddr)
| Type | Sample | Description |
| ---- | ---- |---- |
|12D3KooWLHjy7D | Pure id| Only know id, don't know protocol, ip, etc. |
|/p2p/12D3KooWLHjy7D|pure id | only know id, not protocol, ip, etc.|
|/ip4/1.1.1.1/tcp/4001/p2p/12D3KooWLHjy7D| Detailed path|Know ip, protocol, tcp used|
|/ip4/1.1.1.1/udp/4001/quic-v1/p2p/12D3KooWLHjy7D| Detailed path|Know ip, protocol, use quic|

When the node starts, it will output the corresponding address, just change the ip inside to the public network ip.

You can control the connection behavior through tcp and quic in the path.

### open local port
`./go-p2ptunnel -l 127.0.0.1:3389`

Note that your node id will be output here, and then sent to your friends through chat software. Here, the id is assumed to be 12D3.

The complete ip plus port is required here.

### connect
`./go-p2ptunnel -id 12D3 -l 127.0.0.1:10089`

The connection may take a few seconds to 1 minute. After the connection is successful, the remote port is mapped to 127.0.0.1:10089

Then a friend can connect to 127.0.0.1:10089 on the remote desktop.

### releases

`goreleaser release --skip-publish  --rm-dist`


### Service publishing and sharing(todo)

You can publish your service, and after other nodes search for the service name, they can connect and use it. Must be a tcp-based service, udp is not supported yet.

Service naming `Protocol of this application + / + Protocol of the service name`, if it is not a standard well-known protocol, it is recommended to use a form similar to the package name to avoid service conflicts.


## Precautions
1. Although this application uses end-to-end encryption, it does not guarantee the security of the transmitted data. Please do not use this application to transmit important data.

2. Since it is a p2p tunnel, this program will connect multiple ip, if you mind, please use frp.

3. If there are multiple client connections, please increase the maximum number of files on the server, otherwise the number of connections may not be enough.

## Upstream project

[go-libp2p](https://github.com/libp2p/go-libp2p)