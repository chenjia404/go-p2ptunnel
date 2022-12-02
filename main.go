package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	routing2 "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
)

const Protocol = "/p2ptunnel/0.1"

func loadUserPrivKey() (priv crypto.PrivKey, err error) {
	krPath := "./user.key"
	pkFile, err := os.Open(krPath)

	if err == nil {
		defer pkFile.Close()

		b, err := ioutil.ReadAll(pkFile)
		if err != nil {
			return nil, err
		}

		priv, err = crypto.UnmarshalPrivateKey(b)
		if err != nil {
			return nil, err
		}

		return priv, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	priv, _, err = crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}
	b, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Dir(krPath), os.ModePerm)
	if err != nil {
		return nil, err
	}
	newPkFile, err := os.Create(krPath)
	if err != nil {
		return nil, err
	}
	_, err = newPkFile.Write(b)
	if err != nil {
		return nil, err
	}
	err = newPkFile.Close()
	if err != nil {
		return nil, err
	}

	return priv, nil
}

var d *dht.IpfsDHT

func createLibp2pHost(ctx context.Context, priv crypto.PrivKey) (host.Host, error) {

	connmgr_, _ := connmgr.NewConnManager(
		100,  // Lowwater
		2000, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.UserAgent("go-p2ptunnel"),

		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/udp/0/quic",
			"/ip6/::/udp/0/quic",

			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",

			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip6/::/tcp/0/ws",

			"/ip4/0.0.0.0/udp/0/quic/webtransport",
			"/ip6/::/udp/0/quic/webtransport",
		),

		libp2p.DefaultTransports,
		libp2p.Transport(webtransport.New),

		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		libp2p.NATPortMap(),

		libp2p.ConnectionManager(connmgr_),

		libp2p.EnableRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableAutoRelay(autorelay.WithDefaultStaticRelays(), autorelay.WithCircuitV1Support(), autorelay.WithNumRelays(20)),
		libp2p.DefaultPeerstore,

		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			d, err = dht.New(ctx, h, dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...))
			return d, err
		}),
	)
	if err != nil {
		return nil, err
	}

	// This connects to public bootstrappers
	for _, addr := range dht.DefaultBootstrapPeers {
		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		h.Connect(ctx, *pi)
	}

	err = d.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}
	d1 := routing2.NewRoutingDiscovery(d)

	go func() {
		_, err = d1.Advertise(ctx, Protocol)

		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		peerChan, err := d1.FindPeers(ctx, Protocol)
		if err != nil {
			log.Println(err)
		}

		for peer := range peerChan {
			if peer.ID == h.ID() {
				//log.Println("过滤自己")
				continue
			}

			if h.Network().Connectedness(peer.ID) != network.Connected {
				//log.Println("尝试连接:", peer)
				err = h.Connect(ctx, peer)
				if err == nil {
					//log.Println("连接成功", peer)
					//fmt.Printf("当前连接节点数%d\n", len(h.Network().Peers()))
				} else {
					//log.Println(err)
				}
			}

		}

	}()

	return h, err
}

func main() {
	fmt.Printf("v %d\n", 3)
	fmt.Printf("System version: %s\n", runtime.GOARCH+"/"+runtime.GOOS)
	fmt.Printf("Golang version: %s\n", runtime.Version())

	ip := flag.String("l", "127.0.0.1:10086", "forwarder to ip or listen ip")
	id := flag.String("id", "", "Destination multiaddr id string")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	priv, _ := loadUserPrivKey()

	h, err := createLibp2pHost(ctx, priv)
	if err != nil {
		cancel()
		fmt.Printf("err", err)
		//return nil, nil, err
	}

	fmt.Println("Your id: " + h.ID().String())

	//打开隧道
	if *id == "" {

		h.SetStreamHandler(Protocol, func(s network.Stream) {

			fmt.Println("新客户端\n")
			dconn, err := net.Dial("tcp", *ip)
			if err != nil {
				fmt.Printf("连接%v失败:%v\n", ip, err)
				s.Close()
				return
			} else {
				fmt.Printf("转发:%s\n", *ip)
			}
			go pipe(dconn, s)
		})

	} else {
		//连接指定节点
		// Turn the destination into a multiaddr.
		maddr, err := multiaddr.NewMultiaddr(string("/ipfs/" + *id))
		if err != nil {
			log.Fatalln("multiaddr", err)
		}

		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Panic(err)
		}

		// Add the destination's peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		time.Sleep(time.Second * 5)
		err = h.Connect(ctx, *info)
		if err != nil {
			log.Println("Connect:", err)
		} else {
			fmt.Printf("连接成功", info.ID.String(), "\n")

			lis, err := net.Listen("tcp", *ip)
			if err != nil {
				fmt.Println("Listen:", err)
				return
			} else {
				fmt.Printf("监听:%s\n", *ip)
			}

			go func() {
				for {
					s, err := h.NewStream(ctx, info.ID, Protocol)
					if err != nil {
						fmt.Println(err)
						continue
					}
					conn, err := lis.Accept()
					if err != nil {
						fmt.Println("建立连接错误:%v\n", err)
					} else {
						fmt.Println("新请求")
					}

					go pipe(conn, s)
				}

			}()

		}
	}

	select {}
}
func pipe(src net.Conn, dest network.Stream) {
	errChan := make(chan error, 1)
	onClose := func(err error) {
		fmt.Println("Close")
		_ = dest.Close()
		_ = src.Close()
	}
	go func() {
		_, err := io.Copy(src, dest)
		errChan <- err
		onClose(err)
	}()
	go func() {
		_, err := io.Copy(dest, src)
		errChan <- err
		onClose(err)
	}()
	<-errChan
}
