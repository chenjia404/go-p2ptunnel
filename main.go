package main

import (
	"context"
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

	"github.com/things-go/go-socks5"

	"github.com/chenjia404/go-p2ptunnel/config"
	"github.com/chenjia404/go-p2ptunnel/p2p"

	"github.com/chenjia404/go-p2ptunnel/update"

	"github.com/chenjia404/go-p2ptunnel/pRuntime"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
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

var (
	version   = "0.2.21"
	gitRev    = ""
	buildTime = ""
)

var nodisc bool
var user = "user"

func main() {

	config.LoadConfig()

	if config.Cfg.Update {
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

		if config.Cfg.AutoUpdate {
			go update.CheckGithubVersion(version)
		}
	}

	fmt.Printf("p2ptunnel %s-%s\n", version, gitRev)
	fmt.Printf("buildTime %s\n", buildTime)
	fmt.Printf("System version: %s\n", runtime.GOARCH+"/"+runtime.GOOS)
	fmt.Printf("Golang version: %s\n", runtime.Version())

	if len(config.Cfg.User) > 0 {
		user = config.Cfg.User
	}

	ctx, cancel := context.WithCancel(context.Background())

	priv, _ := loadUserPrivKey()

	h, err := p2p.CreateLibp2pHost(ctx, priv, config.Cfg.P2pPort, config.Cfg.MaxPeers, config.Cfg.Nodisc, Protocol)
	if err != nil {
		cancel()
		fmt.Printf("err:%s", err.Error())
		//return nil, nil, err
	}

	fmt.Println("Your id: " + h.ID().String())
	if nodisc {
		fmt.Println("Turn off node discovery")
	}

	if len(config.Cfg.Socks5) >= 6 {
		server := socks5.NewServer(
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
		)

		// Create SOCKS5 proxy on localhost port 8000
		go func() {
			if err := server.ListenAndServe("tcp", config.Cfg.Socks5); err != nil {
				panic(err)
			}
		}()
		log.Printf("socks5 open:%s\n", config.Cfg.Socks5)
	}

	//打开隧道
	if config.Cfg.Id == "" {

		ticker := time.NewTicker(time.Second * 10)
		go func() {
			for { // 用上一个死循环，不停地执行，否则只会执行一次
				select {
				case <-ticker.C:
					log.Printf("Conns:%d\n", len(h.Network().Conns()))
				}
			}
		}()

		h.SetStreamHandler(Protocol, func(s network.Stream) {
			log.Printf("新客户端%s\n", s.Conn().RemotePeer().String())
			dconn, err := net.Dial("tcp", config.Cfg.Listen)
			if err != nil {
				fmt.Printf("连接%v失败:%v\n", config.Cfg.Listen, err)
				s.Close()
				return
			} else {
				fmt.Printf("转发:%s\n", config.Cfg.Listen)
				fmt.Printf("Streams:%d\n", len(s.Conn().GetStreams()))

			}
			go pipe(dconn, s)
		})

	} else {
		//连接指定节点
		// Turn the destination into a multiaddr.
		id_str := config.Cfg.Id
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

		lis, err := net.Listen("tcp", config.Cfg.Listen)
		if err != nil {
			fmt.Println("Listen:", err)
			return
		} else {
			fmt.Printf("监听:%s\n", config.Cfg.Listen)
		}

		for {
			h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
			err = h.Connect(ctx, *info)
			if err != nil {
				log.Println("Connect:", err)
				time.Sleep(time.Second * 5)
			} else {
				fmt.Printf("连接成功%s\n", info.ID.String())
				for {
					if s != nil {

						log.Printf("Stream:%d\n", len(s.Conn().GetStreams()))
						if len(s.Conn().GetStreams()) == 0 {
							err = h.Connect(ctx, *info)
							if err != nil {
								log.Println("Connect:", err)
								time.Sleep(time.Second * 5)
							}
						}
					}
					log.Println("open New Stream")
					timeout, _ := context.WithTimeout(context.Background(), 5*time.Second)

					s, err = h.NewStream(timeout, info.ID, Protocol)
					if err != nil {
						fmt.Println("New Stream:" + err.Error())
						err = h.Connect(ctx, *info)
						if err != nil {
							log.Println("Connect:", err)
							time.Sleep(time.Second * 5)
						}
					} else {
						log.Println("New Stream is open")
					}

					// 长时间休眠，已经没有 Stream 了
					if s == nil || len(s.Conn().GetStreams()) == 0 {
						continue
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
	onClose := func(err error) {
		_ = dest.Reset()
		_ = src.Close()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.Copy(src, dest)
		src.SetReadDeadline(time.Now().Add(wait)) // unblock read on right
		onClose(err)
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(dest, src)
		dest.SetReadDeadline(time.Now().Add(wait)) // unblock read on left
		onClose(err)
	}()
	wg.Wait()
}
