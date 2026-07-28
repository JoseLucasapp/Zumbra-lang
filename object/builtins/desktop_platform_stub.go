//go:build !linux || !cgo

package builtins

import (
	"os"
	"os/exec"
	"sync"

	"zumbra/object"
)

func newPlatformDesktopBackend(options map[string]object.Object) (object.DesktopBackend, error) {
	return newHeadlessDesktopBackend(), nil
}

type osDesktopProcess struct {
	mu      sync.RWMutex
	cmd     *exec.Cmd
	done    chan struct{}
	exit    int64
	err     error
	running bool
	pid     int64
}

func startDesktopProcess(command string, args []string, options map[string]object.Object) (object.DesktopProcessRuntime, error) {
	cmd := exec.Command(command, args...)
	if cwd := optionString(options, "cwd", ""); cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &osDesktopProcess{cmd: cmd, done: make(chan struct{}), running: true, pid: int64(cmd.Process.Pid)}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.err = err
		p.running = false
		if cmd.ProcessState != nil {
			p.exit = int64(cmd.ProcessState.ExitCode())
		}
		p.mu.Unlock()
		close(p.done)
	}()
	return p, nil
}
func (p *osDesktopProcess) PID() int64 { return p.pid }
func (p *osDesktopProcess) Wait() (int64, error) {
	<-p.done
	p.mu.RLock()
	defer p.mu.RUnlock()
	if _, ok := p.err.(*exec.ExitError); ok {
		return p.exit, nil
	}
	return p.exit, p.err
}
func (p *osDesktopProcess) Kill() error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()
	if !running {
		return nil
	}
	return p.cmd.Process.Kill()
}
func (p *osDesktopProcess) Running() bool { p.mu.RLock(); defer p.mu.RUnlock(); return p.running }
func parseProcessArgs(value object.Object) ([]string, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("desktopSpawn arguments must be array")
	}
	result := make([]string, len(array.Elements))
	for index, item := range array.Elements {
		text, ok := item.(*object.String)
		if !ok {
			return nil, NewError("desktopSpawn argument %d must be string", index)
		}
		result[index] = text.Value
	}
	return result, nil
}
