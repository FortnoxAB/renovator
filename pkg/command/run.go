package command

import (
	"os"
	"os/exec"
	"strings"
)

type Commander interface {
	Run(head string, parts ...string) (stdOut, stdErr string, exitCode int, err error)
	RunWithEnv(env []string, head string, parts ...string) (stdOut, stdErr string, exitCode int, err error)
}

type Exec struct {
}

func (e *Exec) Run(head string, parts ...string) (stdOut, stdErr string, exitCode int, err error) {
	return e.RunWithEnv(nil, head, parts...)
}

func (e *Exec) RunWithEnv(env []string, head string, parts ...string) (stdOut, stdErr string, exitCode int, err error) {
	cmd := exec.Command(head, parts...) // #nosec
	outBuf := &strings.Builder{}
	errBuf := &strings.Builder{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	err = cmd.Run()
	exitCode = cmd.ProcessState.ExitCode()
	stdOut = strings.TrimSpace(outBuf.String())
	stdErr = strings.TrimSpace(errBuf.String())
	return
}
