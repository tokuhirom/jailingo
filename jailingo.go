package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const VERSION = "0.0.1"

type stringArray []string

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
	Root     string
	Bind     []string
	TempDirs []string
	Devices  []*Device
}

func (i *stringArray) String() string {
	return "stringArray"
}

func (i *stringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func Copy(dst, src string) error {
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
	}
}

func (app *JailingApp) copyFiles(copy_files []string) error {
	// mkdir etc/
	err := os.MkdirAll(filepath.Join(app.Root, "/etc/"), 0755)
	if err != nil {
		log.Fatal("mkdir /etc/ ", err)
	}

	// Copy files
	for _, filename := range copy_files {
		if _, err := os.Stat(filepath.Join("/", filename)); os.IsNotExist(err) {
			log.Debug("No file: ", filename)
		} else {
			log.Debug("Copy file: ", filename)
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
			log.Debug("Creating new device: ", device)
			app.mknod(device.Path, device.Mode, device.Major, device.Minor)
		} else {
			log.Debug("Device exists: ", device)
		}
	}
	return nil
}

func (app *JailingApp) createSymlinks() {
	err := os.Symlink("../run/lock", filepath.Join(app.Root, "var/lock"))
	if err != nil {
		log.Fatal(err)
	}
}

func (app *JailingApp) Main() {
	// Step in to root directory
	err := os.Chdir(app.Root)
	if err != nil {
		log.Fatal("Chdir ", err)
	}

	err = os.MkdirAll(app.Root, 0755)
	if err != nil {
		log.Fatal("mkdirs ", err)
	}

	for _, tmpdir := range app.TempDirs {
		path := filepath.Join(app.Root, tmpdir)
		err = os.MkdirAll(path, 01777)
		if err != nil {
			log.Fatal("mkdirs ", err)
		}
		err = os.Chmod(path, 01777)
		if err != nil {
			log.Fatal("chmod ", err)
		}
	}

	app.createSymlinks()

	// make devices
	app.makeDevices()

	// copy files
	copy_files := []string{"etc/group",
		"etc/passwd",
		"etc/resolv.conf",
		"etc/hosts"}
	app.copyFiles(copy_files)

	// TOOD mount bind

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
}

func main() {
	root := flag.String("root", "", "chroot root")
	levelString := flag.String("log.level", "INFO", "log level")
	var binds stringArray
	flag.Var(&binds, "bind", "binds")
	version := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *root == "" {
		log.Fatal("Missing --root argument")
	}
	if (*root)[0] != '/' {
		log.Fatal("--root argument must be absolute")
	}
	if *version {
		fmt.Printf("%v\n", VERSION)
		return
	}

	level, err := log.ParseLevel(*levelString)
	if err != nil {
		log.Fatal("Invalid logging level: ", err)
	}
	log.SetLevel(level)

	app := NewJailingApp(*root, binds)
	app.Main()
}
