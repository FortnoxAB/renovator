package command

import (
	"os"
	"os/exec"
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

	err = cmd.Run()
	return
}
