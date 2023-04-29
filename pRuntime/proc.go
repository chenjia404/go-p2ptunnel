package pRuntime

import (
	"fmt"
	"os"
)

type Proc struct {
	proc      *os.Process
	procState *os.ProcessState
}

func NewProc() (*Proc, error) {
	if os.Getenv("__NewProc") == "true" {
		return nil, nil
	}
	cmdRet, err := forkDaemon(false, "__NewProc=true")
	if err != nil {
		return nil, err
	}
	return &Proc{
		proc: cmdRet.Process,
	}, nil
}

func (p *Proc) Pid() int {
	if p.proc == nil {
		return 0
	}
	return p.proc.Pid
}

func (p *Proc) Kill() error {
	if p.proc == nil {
		return fmt.Errorf("proc is null")
	}
	return p.proc.Kill()
}

func (p *Proc) Wait() error {
	var err error
	p.procState, err = p.proc.Wait()
	if err != nil {
		return err
	}
	return nil
}
