package core

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Device struct {
	Path  string
	Mode  int
	Major int
	Minor int
}

func NewDevice(path string, mode, major, minor int) *Device {
	return &Device{path, mode, major, minor}
}

type JailingApp struct {
	Root      string
	Bind      []string
	TempDirs  []string
	Devices   []*Device
	CopyFiles []string
}

func Copy(dst, src string) error {
	log.Info("Copy ", src, " to ", dst)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func NewJailingApp(root string, binds []string) *JailingApp {
	return &JailingApp{
		root,
		binds,
		[]string{"tmp", "run/lock", "var/tmp"},
		[]*Device{
			NewDevice("/dev/null", 0666, 1, 3),
			NewDevice("/dev/zero", 0666, 1, 5),
			NewDevice("/dev/random", 0666, 1, 9),
			NewDevice("/dev/urandom", 0666, 1, 9),
		},
		[]string{
			"etc/group",
			"etc/passwd",
			"etc/resolv.conf",
			"etc/hosts",
		},
	}
}

func (app *JailingApp) copyFiles() error {
	for _, filename := range app.CopyFiles {
		dst := filepath.Join(app.Root, filename)
		src := filepath.Join("/", filename)
		err := Copy(dst, src)
		if err != nil {
			return fmt.Errorf("Cannot copy %v to %v: %v", src, dst, err)
		}
	}
	return nil
}

func MakeDev(major, minor int) int {
	/*
		a := minor & 0xff
		b := (major & 0xfff) << 8
		c := ((int(minor) & ^0xff) << 12)
		d := ((int(major) & ^0xfff) << 32)
		return a | b | c | d
	*/
	return major*256 + minor
}

func (app *JailingApp) mknod(path string, mode int, major int, minor int) error {
	if _, err := os.Stat(filepath.Join(app.Root, path)); os.IsNotExist(err) {
		log.Info("Creating new device: ", path)
		err = syscall.Mknod(filepath.Join(app.Root, path), syscall.S_IFCHR|uint32(mode), MakeDev(major, minor))
		if err != nil {
			return fmt.Errorf("Cannot make device: ", path, err)
		}
	} else {
		log.Info("Device exists: ", path)
	}
	return nil
}

func (app *JailingApp) makeDevices() error {
	err := os.MkdirAll(filepath.Join(app.Root, "/dev/"), 0755)
	if err != nil {
		log.Fatal("mkdir /dev/ ", err)
	}
	for _, device := range app.Devices {
		err = app.mknod(device.Path, device.Mode, device.Major, device.Minor)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *JailingApp) createSymlink(target, link_name string) error {
	if _, err := os.Stat(link_name); os.IsNotExist(err) {
		err := os.Symlink(target, link_name)
		if err != nil {
			return err
		}
		return nil
	} else {
		log.Infof("%v already exists", link_name)
		return nil
	}
}

func (app *JailingApp) createSymlinks() {
	err := app.createSymlink("../run/lock", filepath.Join(app.Root, "var/lock"))
	if err != nil {
		log.Fatal(err)
	}
}

func (app *JailingApp) MakeTempDirs() error {
	for _, tmpdir := range app.TempDirs {
		path := filepath.Join(app.Root, tmpdir)
		err := os.MkdirAll(path, 01777)
		if err != nil {
			return err
		}
		err = os.Chmod(path, 01777)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *JailingApp) Main() error {
	err := os.MkdirAll(app.Root, 0755)
	if err != nil {
		return err
	}

	// Step in to root directory
	err = os.Chdir(app.Root)
	if err != nil {
		return err
	}

	app.MakeTempDirs()

	app.createSymlinks()

	// make devices
	app.makeDevices()

	// mkdir /etc/
	err = os.MkdirAll(filepath.Join(app.Root, "/etc/"), 0755)
	if err != nil {
		return err
	}

	// copy files
	err = app.copyFiles()
	if err != nil {
		return err
	}

	// TODO make /dev as tmpfs. since some env provides root fs as 'nodev'
	// TOOD mount bind
	// TODO defer umount

	// Do chroot
	err = syscall.Chroot(app.Root)
	if err != nil {
		return fmt.Errorf("Cannot chroot: ", app.Root, err)
	}

	// TODO drop_capabilities

	// Execute command
	cmd := exec.Command("/bin/sh")
	err = cmd.Start()
	if err != nil {
		log.Fatal("executing command: ", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
