package config

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var ip = flag.String("l", "", "forwarder to ip or listen ip")
var id = flag.String("id", "", "Destination multiaddr id string")
var p2p_port = flag.Int("p2p_port", 4001, "p2p use port")
var max_peers = flag.Int("max_peers", 500, "Maximum number of connections, default 500")
var flag_nodisc = flag.Bool("nodisc", false, "Turn off node discovery")
var flag_user = flag.String("user", "user", "Turn off node discovery")
var networkType = flag.String("type", "tcp", "network type tcp/udp")
var flag_update = flag.Bool("update", false, "update form github")
var flag_auto_update = flag.Bool("auto_update", false, "update form github")
var configPath = flag.String("config", "", "config file")
var socks5 = flag.String("socks5", "", "socks5 listen ip")

var Cfg *Conf

type Conf struct {
	User       string
	Listen     string
	Socks5     string
	Id         string
	P2pPort    int
	MaxPeers   int
	Nodisc     bool
	Update     bool
	AutoUpdate bool
	ConfigPath string
}

func init() {
	flag.Parse()
}

func LoadConfig() error {
	return LoadConfigByPath(*configPath)
}

func LoadConfigByPath(p string) error {
	_, err := os.Stat("config.yaml")
	if err == nil && p == "" {
		p = "./config.yaml"
	}
	if p == "" {
		Cfg = &Conf{
			User:     *flag_user,
			Listen:   *ip,
			Socks5:   *socks5,
			Id:       *id,
			P2pPort:  *p2p_port,
			MaxPeers: *max_peers,
			Nodisc:   *flag_nodisc,
			Update:   *flag_update,
		}
		if len(Cfg.Socks5) >= 6 && len(Cfg.Listen) == 0 {
			Cfg.Listen = Cfg.Socks5
		}
		return nil
	}

	Cfg = &Conf{}
	paths, fileName := filepath.Split(p)
	viper.SetConfigName(fileName)
	viper.AddConfigPath(paths)
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	Cfg.User = viper.GetString("key.user")
	Cfg.Listen = viper.GetString("net.listen")
	Cfg.Socks5 = viper.GetString("net.socks5")
	Cfg.Id = viper.GetString("net.id")
	Cfg.P2pPort = viper.GetInt("net.p2p_port")
	Cfg.MaxPeers = viper.GetInt("net.max_peers")
	Cfg.Nodisc = viper.GetBool("net.no_disc")
	Cfg.Update = *flag_update
	Cfg.AutoUpdate = *flag_auto_update
	if len(Cfg.Socks5) >= 6 && len(Cfg.Listen) == 0 {
		Cfg.Listen = Cfg.Socks5
	}
	return nil
}
