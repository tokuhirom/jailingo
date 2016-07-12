package core

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"syscall"
)

type Unmounter struct {
	Root    string
	Binds   []string
	RoBinds []string
}

func NewUnmounter(root string, binds []string, robinds []string) *Unmounter {
	return &Unmounter{
		root,
		binds,
		robinds,
	}
}

func (app *Unmounter) Unmount(mount string) error {
	target := filepath.Join(app.Root, mount)
	if IsEmpty(target) {
		log.Infof("%s is empty", target)
		return nil
	}

	log.Infof("Unmounting %s", target)
	err := syscall.Unmount(target, syscall.MNT_DETACH)
	if err != nil {
		/*

			EINVAL target is not a mount point.

			EINVAL umount2() was called with MNT_EXPIRE and either MNT_DETACH or
					MNT_FORCE.

			EINVAL (since Linux 2.6.34)
					umount2() was called with an invalid flag value in flags.

		*/
		return fmt.Errorf("Cannout unmount %v: %v", target, err)
	}
	return nil
}

func (app *Unmounter) UnmountAll() error {
	err := app.Unmount("/dev")
	if err != nil {
		return err
	}

	for _, mount := range app.Binds {
		err := app.Unmount(mount)
		if err != nil {
			return err
		}
	}
	for _, mount := range app.RoBinds {
		err := app.Unmount(mount)
		if err != nil {
			return err
		}
	}
	return nil
}
