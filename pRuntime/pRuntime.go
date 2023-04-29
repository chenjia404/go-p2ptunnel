package pRuntime

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

var pidFile = "go-p2ptunnel.pid"

func SetPidFile(pFile string) {
	pidFile = pFile
}

func forkDaemon(isWritePidFile bool, environ ...string) (*exec.Cmd, error) {
	cmdRet := &exec.Cmd{
		Path:   os.Args[0],
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    append(os.Environ(), environ...),
	}
	err := cmdRet.Start()
	if err != nil {
		return nil, err
	}
	//写入pid
	if isWritePidFile {
		err = ioutil.WriteFile(pidFile, []byte(strconv.Itoa(cmdRet.Process.Pid)), 0666)
	}
	if err != nil {
		return nil, err
	}
	return cmdRet, nil
}

func DaemonInit() {
	if runtime.GOOS == "windows" {
		return
	}
	if os.Getenv("__Daemon") == "true" {
		return
	}
	c := "start"
	if l := len(os.Args); l > 1 {
		c = os.Args[l-1]
	}
	switch c {
	case "start":
		if CheckProIsRun() {
			log.Fatal("当前进程已运行...")
		}
		cmdRet, err := forkDaemon(true, "__Daemon=true")
		if err != nil {
			log.Fatal("start err : ", err)
		}
		log.Println("Daemon is run... pid: ", cmdRet.Process.Pid)
	case "restart":
		err := Stop()
		if err != nil {
			log.Fatal("restart stop err : ", err)
		}
		os.Args = os.Args[:len(os.Args)-1]
		cmdRet, err := forkDaemon(true, "__Daemon=true")
		if err != nil {
			log.Fatal("forkDaemon err : ", err)
		}
		log.Println("Daemon is run... pid: ", cmdRet.Process.Pid)
	case "stop":
		err := Stop()
		if err != nil {
			log.Fatal("stop err : ", err)
		}
		log.Println("Daemon is stop....")
	case "reload":
		err := Reload()
		if err != nil {
			log.Fatal("reload err : ", err)
		}
		log.Println("Daemon reload success....")
	}
	os.Exit(0)
}

func CheckProIsRun() bool {
	if GetRunningPid() == 0 {
		return false
	}
	return true
}

func FileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func GetRunningPid() int {
	if !FileExists(pidFile) {
		return 0
	}
	b, err := ioutil.ReadFile(pidFile)
	if err != nil {
		log.Fatal("程序异常：", err)
	}
	pid, _ := strconv.Atoi(string(b))
	p, err := os.FindProcess(pid)
	if err != nil {
		return 0
	}
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return p.Pid
	}
	_ = os.Remove(pidFile)
	return 0
}

func HandleEndSignal(fn func()) {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	_ = os.Remove(pidFile)
	fn()
	return
}

func HandleReloadSignal(fn func()) {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	for {
		<-sig
		fn()
	}
}

func Stop() error {
	if !CheckProIsRun() {
		return errors.New("进程没有运行")
	}
	pro, err := os.FindProcess(GetRunningPid())
	if err != nil {
		return err
	}
	return pro.Signal(syscall.SIGTERM)
}

func Reload() error {
	if !CheckProIsRun() {
		return errors.New("进程没有运行")
	}
	pro, err := os.FindProcess(GetRunningPid())
	if err != nil {
		return err
	}
	return pro.Signal(syscall.SIGHUP)
}
