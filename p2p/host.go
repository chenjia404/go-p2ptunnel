package p2p

import (
	"context"
	"fmt"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	routing2 "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

var d *dht.IpfsDHT

func CreateLibp2pHost(ctx context.Context, priv crypto.PrivKey, p2pPort int, maxPeers int, nodisc bool, Protocol string) (host.Host, error) {

	connmgr_, _ := connmgr.NewConnManager(
		10,       // Lowwater
		maxPeers, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)

	wsPort := p2pPort + 1
	if p2pPort == 0 {
		wsPort = 0
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.UserAgent("go-p2ptunnel"),

		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort),
			fmt.Sprintf("/ip6/::/tcp/%d", p2pPort),

			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", wsPort),
			fmt.Sprintf("/ip6/::/tcp/%d/ws", wsPort),

			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", p2pPort),
			fmt.Sprintf("/ip6/::/udp/%d/quic-v1", p2pPort),

			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1/webtransport", p2pPort),
			fmt.Sprintf("/ip6/::/udp/%d/quic-v1/webtransport", p2pPort),
		),

		libp2p.DefaultTransports,

		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		libp2p.ConnectionManager(connmgr_),

		libp2p.NATPortMap(),

		libp2p.EnableRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
		libp2p.DefaultPeerstore,

		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			if !nodisc {
				var err error
				d, err = dht.New(ctx, h, dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...))
				return d, err
			}
			return nil, nil
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
		_, h2, err2 := nodeDiscovery(ctx, h, Protocol)
		if err2 != nil {
			return h2, err2
		}

	}

	for _, value := range h.Addrs() {
		fmt.Println("multiaddr:" + value.String())
	}

	return h, err
}

func nodeDiscovery(ctx context.Context, h host.Host, Protocol string) (error, host.Host, error) {
	err := d.Bootstrap(ctx)
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
	return err, h, nil
}
