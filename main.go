package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/chenjia404/go-p2ptunnel/update"

	"github.com/chenjia404/go-p2ptunnel/pRuntime"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	routing2 "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

const Protocol = "/p2ptunnel/0.1"

func loadUserPrivKey() (priv crypto.PrivKey, err error) {
	krPath := user + ".key"
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

func createLibp2pHost(ctx context.Context, priv crypto.PrivKey, p2p_port int) (host.Host, error) {

	connmgr_, _ := connmgr.NewConnManager(
		10,  // Lowwater
		200, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.UserAgent("go-p2ptunnel"),

		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2p_port),
			fmt.Sprintf("/ip6/::/tcp/%d", p2p_port),

			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", p2p_port),
			fmt.Sprintf("/ip6/::/tcp/%d/ws", p2p_port),

			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", p2p_port),
			fmt.Sprintf("/ip6/::/udp/%d/quic-v1", p2p_port),

			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1/webtransport", p2p_port),
			fmt.Sprintf("/ip6/::/udp/%d/quic-v1/webtransport", p2p_port),
		),

		libp2p.DefaultTransports,

		libp2p.Security(noise.ID, noise.New),

		libp2p.ConnectionManager(connmgr_),

		libp2p.NATPortMap(),

		libp2p.EnableRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
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

	if !nodisc {
		_, h2, err2 := nodeDiscovery(ctx, err, h)
		if err2 != nil {
			return h2, err2
		}

	}

	for _, value := range h.Addrs() {
		fmt.Println("multiaddr:" + value.String())
	}

	return h, err
}

func nodeDiscovery(ctx context.Context, err error, h host.Host) (error, host.Host, error) {
	err = d.Bootstrap(ctx)
	if err != nil {
		return nil, nil, err
	}
	d1 := routing2.NewRoutingDiscovery(d)

	go func() {
		_, err = d1.Advertise(ctx, Protocol)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {

		for i := 0; i < 10; {
			// log.Println("开始寻找节点")
			_, err = d1.Advertise(ctx, Protocol)

			if err != nil {
				log.Println(err)
			}

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
						// log.Println("连接成功", peer.ID)
						// fmt.Printf("当前连接节点数%d\n", len(h.Network().Peers()))
						i++
					} else {
						//log.Println(err)
					}
				}

			}
		}

	}()
	return err, nil, nil
}

var (
	version   = "0.0.9"
	gitRev    = ""
	buildTime = ""
)

var nodisc bool
var user = "user"

func main() {

	ip := flag.String("l", "127.0.0.1:10086", "forwarder to ip or listen ip")
	id := flag.String("id", "", "Destination multiaddr id string")
	p2p_port := flag.Int("p2p_port", 4001, "p2p use port")
	flag_nodisc := flag.Bool("nodisc", false, "Turn off node discovery")
	flag_user := flag.String("user", "user", "Turn off node discovery")
	networkType := flag.String("type", "tcp", "network type tcp/udp")
	flag_update := flag.Bool("update", false, "update form github")

	flag.Parse()

	if *flag_update {
		update.CheckGithubVersion(version)
		return
	}
RE:
	proc, err := pRuntime.NewProc()
	if err != nil {
		fmt.Println("up proc fail........")
	}
	//如果proc为nil表示当前进程已经是子进程了
	//不为空表示当前进程为主进程
	if proc != nil {
		go func() {
			pRuntime.HandleEndSignal(func() {
				if err := proc.Kill(); err != nil {
					fmt.Println(err)
				}
				fmt.Println("main proc exit....")

				os.Exit(0)
			})
		}()
		//等待子进程退出后 重启
		err = proc.Wait()
		if err != nil {
			fmt.Println("proc wait err........")
		} else {
			goto RE
		}
		return
	} else {

		go func() {
			now := time.Now()
			next := now.Add(time.Hour * 4)
			timer := time.NewTimer(next.Sub(now))
			t := <-timer.C //从定时器拿数据
			fmt.Println("restart time:", t)
			os.Exit(0)
		}()

	}

	fmt.Printf("p2ptunnel %s-%s\n", version, gitRev)
	fmt.Printf("buildTime %s\n", buildTime)
	fmt.Printf("System version: %s\n", runtime.GOARCH+"/"+runtime.GOOS)
	fmt.Printf("Golang version: %s\n", runtime.Version())

	nodisc = *flag_nodisc
	if len(*flag_user) > 0 {
		user = *flag_user
	}

	ctx, cancel := context.WithCancel(context.Background())

	priv, _ := loadUserPrivKey()

	h, err := createLibp2pHost(ctx, priv, *p2p_port)
	if err != nil {
		cancel()
		fmt.Printf("err", err)
		//return nil, nil, err
	}

	fmt.Println("Your id: " + h.ID().String())
	if nodisc {
		fmt.Println("Turn off node discovery")
	}

	//打开隧道
	if *id == "" {

		ticker := time.NewTicker(time.Second * 10)
		go func() {
			for { // 用上一个死循环，不停地执行，否则只会执行一次
				select {
				case <-ticker.C:
					log.Printf("Stream:%d\n", len(h.Network().Conns()))
				}
			}
		}()

		h.SetStreamHandler(Protocol, func(s network.Stream) {
			log.Printf("新客户端%s\n", s.Conn().RemotePeer().String())
			dconn, err := net.Dial(*networkType, *ip)
			if err != nil {
				fmt.Printf("连接%v失败:%v\n", ip, err)
				s.Close()
				return
			} else {
				fmt.Printf("转发:%s\n", *ip)
				fmt.Printf("Conns:%d\n", len(h.Network().Conns()))

			}
			go pipe(dconn, s)
		})

	} else {
		//连接指定节点
		// Turn the destination into a multiaddr.
		id_str := string(*id)
		if id_str[0] != '/' {
			id_str = "/p2p/" + id_str
		}

		maddr, err := multiaddr.NewMultiaddr(id_str)

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
		//time.Sleep(time.Second * 5)
		var s network.Stream
		ticker := time.NewTicker(time.Second * 10)
		go func() {
			for { // 用上一个死循环，不停地执行，否则只会执行一次
				select {
				case <-ticker.C:
					if s != nil {
						log.Printf("Stream:%d\n", len(s.Conn().GetStreams()))
					}
				}
			}
		}()

		lis, err := net.Listen(*networkType, *ip)
		if err != nil {
			fmt.Println("Listen:", err)
			return
		} else {
			fmt.Printf("监听:%s\n", *ip)
		}

		for {
			h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
			err = h.Connect(ctx, *info)
			if err != nil {
				log.Println("Connect:", err)
				time.Sleep(time.Second * 10)
			} else {
				fmt.Printf("连接成功%s\n", info.ID.String())
				for {
					if s != nil {

						log.Printf("Stream:%d\n", len(s.Conn().GetStreams()))
						if len(s.Conn().GetStreams()) == 0 {
							err = h.Connect(ctx, *info)
							if err != nil {
								log.Println("Connect:", err)
								time.Sleep(time.Second * 10)
							}
						}
					}
					log.Println("open New Stream")
					s, err = h.NewStream(ctx, info.ID, Protocol)
					if err != nil {
						fmt.Println("New Stream:" + err.Error())
						err = h.Connect(ctx, *info)
						if err != nil {
							log.Println("Connect:", err)
							time.Sleep(time.Second * 10)
						}
					} else {
						log.Println("New Stream is open")
					}
					conn, err := lis.Accept()
					if err != nil {
						fmt.Printf("建立连接错误:%s\n", err.Error())
					} else {
						fmt.Println("新请求")
					}

					go pipe(conn, s)
				}

			}
		}

	}

	select {}
}
func pipe(src net.Conn, dest network.Stream) {
	var wg sync.WaitGroup
	var wait = 10 * time.Second
	errChan := make(chan error, 1)
	onClose := func(err error) {
		_ = dest.Close()
		_ = src.Close()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(src, dest)
		dest.SetReadDeadline(time.Now().Add(wait)) // unblock read on right
		errChan <- err
		onClose(err)
	}()
	go func() {
		_, err := io.Copy(dest, src)
		errChan <- err
		onClose(err)
	}()
	<-errChan
	src.SetReadDeadline(time.Now().Add(wait)) // unblock read on left
	wg.Wait()
}
