package command

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

type Commander interface {
	Run(head string, parts ...string) error
	RunWithEnv(env []string, head string, parts ...string) error
}

type Exec struct {
}

func (e *Exec) Run(head string, parts ...string) (err error) {
	return e.RunWithEnv(nil, head, parts...)
}

func (e *Exec) RunWithEnv(env []string, head string, parts ...string) (err error) {
	cmd := exec.Command(head, parts...) // #nosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	err = cmd.Start()
	if err != nil {
		return err
	}

	logrus.Debugf("started pid %d, %s", cmd.Process.Pid, head+" "+strings.Join(parts, " "))

	err = cmd.Wait()

	e.reapChildren()
	return err
}

func (e *Exec) reapChildren() {
	for {
		var wstatus syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
		if errors.Is(err, syscall.ECHILD) {
			return
		}
		if err != nil {
			logrus.Errorf("error waiting for child %d: %s", pid, err)
			continue
		}
		if pid == 0 && wstatus == 0 {
			logrus.Debug("found no pid")
			return
		}
		logrus.Debugf("reaped zombie %d %d", pid, wstatus)
	}
}
