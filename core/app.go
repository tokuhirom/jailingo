package core

import (
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
	// mkdir etc/
	err := os.MkdirAll(filepath.Join(app.Root, "/etc/"), 0755)
	if err != nil {
		log.Fatal("mkdir /etc/ ", err)
	}

	// Copy files
	for _, filename := range app.CopyFiles {
		if _, err := os.Stat(filepath.Join("/", filename)); os.IsNotExist(err) {
			log.Info("No file: ", filename)
		} else {
			log.Info("Copy file: ", filename)
			err := Copy(filepath.Join(app.Root, filename), filepath.Join("/", filename))
			if err != nil {
				log.Fatal("Cannot copy file: ", filename, " ", err)
			}
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
		err = syscall.Mknod(filepath.Join(app.Root, path), syscall.S_IFCHR|uint32(mode), MakeDev(major, minor))
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *JailingApp) makeDevices() error {
	err := os.MkdirAll(filepath.Join(app.Root, "/dev/"), 0755)
	if err != nil {
		log.Fatal("mkdir /dev/ ", err)
	}
	for _, device := range app.Devices {
		if _, err := os.Stat(filepath.Join(app.Root, device.Path)); os.IsNotExist(err) {
			log.Info("Creating new device: ", device)
			app.mknod(device.Path, device.Mode, device.Major, device.Minor)
		} else {
			log.Info("Device exists: ", device)
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
	// Step in to root directory
	err := os.Chdir(app.Root)
	if err != nil {
		return err
	}

	err = os.MkdirAll(app.Root, 0755)
	if err != nil {
		return err
	}

	app.MakeTempDirs()

	app.createSymlinks()

	// make devices
	app.makeDevices()

	// copy files
	app.copyFiles()

	// TOOD mount bind
	// TODO defer umount

	// Do chroot
	err = syscall.Chroot(app.Root)
	if err != nil {
		log.Fatal("Cannot chroot ", err)
	}

	// TODO drop_capabilities

	// Execute command
	cmd := exec.Command("/bin/sh")
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
