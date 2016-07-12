package child

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

type Child struct {
	Root     string
	Command  string
	Args     []string
	logLevel string
}

func NewChildProcfs(root string, command string, args []string, logLevel string) *Child {
	return &Child{
		root,
		command,
		args,
		logLevel,
	}
}

func (self *Child) Run() error {
	if err := os.Chdir(self.Root); err != nil {
		return err
	}
	if err := syscall.Chroot(self.Root); err != nil {
		return err
	}

	log.Info("Run")
	// mount procfs
	if err := os.MkdirAll("/proc", 0755); err != nil {
		return err
	}

	log.Infof("Mounting /proc")
	err := syscall.Mount("proc", "/proc", "proc", syscall.MS_MGC_VAL, "")
	if err != nil {
		log.Fatalf("Cannot mount procfs: %v", err)
	}

	cmd := exec.Command(self.Command, self.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Infof("Unmounting /proc")
	if err := syscall.Unmount("/proc", 0); err != nil {
		return err
	}

	return nil
}
