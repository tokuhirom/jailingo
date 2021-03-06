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
	TempDirs  []string
	Devices   []*Device
	CopyFiles []string
	Binds     []string
	RoBinds   []string
	Command   string
	Args      []string
	logLevel  string
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

func NewJailingApp(root string, tmpdirs []string, copyfiles []string, binds []string, robinds []string, command string, args []string, logLevel string) *JailingApp {
	return &JailingApp{
		root,
		tmpdirs,
		[]*Device{
			NewDevice("/dev/null", 0666, 1, 3),
			NewDevice("/dev/zero", 0666, 1, 5),
			NewDevice("/dev/random", 0666, 1, 9),
			NewDevice("/dev/urandom", 0666, 1, 9),
		},
		copyfiles,
		binds,
		robinds,
		command,
		args,
		logLevel,
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

func (app *JailingApp) mknod(path string, mode, major, minor int) error {
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
	devpath := filepath.Join(app.Root, "/dev/")
	err := os.MkdirAll(devpath, 0755)
	if err != nil {
		log.Fatalf("mkdir %v: %v", devpath, err)
	}

	// Make /dev/ as tmpfs.
	// In some case, root fs was mounted with 'nodev' option.
	if IsEmpty(devpath) {
		err = syscall.Mount("tmpfs", devpath, "tmpfs", syscall.MS_MGC_VAL, "")
		if err != nil {
			log.Fatalf("Cannot mount %v as tmpfs: %v", devpath, err)
		}
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

func IsEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true
	}
	return false
}

func (app *JailingApp) mount(point string, readonly bool) error {
	dest := filepath.Join(app.Root, point)

	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	if IsEmpty(dest) {
		log.Infof("Mounting %v(readonly: %v)", point, readonly)
		// sudo strace mount --bind /bin /tmp/jail/bin/
		// mount("/usr/bin", "/tmp/jail/bin", 0x7fc44d050240, MS_MGC_VAL|MS_BIND, NULL) = 0
		// MS_MGC_VAL is required by linux kernel 2.4-
		err = syscall.Mount(point, dest, "bind", syscall.MS_MGC_VAL|syscall.MS_BIND, "")
		if err != nil {
			return fmt.Errorf("Cannot mount(%v): %v", point, err)
		}

		err = syscall.Mount(point, dest, "", syscall.MS_MGC_VAL|syscall.MS_BIND|syscall.MS_RDONLY|syscall.MS_REMOUNT, "")
		if err != nil {
			return fmt.Errorf("Cannot mount(%v): %v", point, err)
		}
	} else {
		log.Infof("%v is mounted(readonly: %v)", point, readonly)
	}

	return nil
}

func (app *JailingApp) path(rel string) string {
	return filepath.Join(app.Root, rel)
}

func (app *JailingApp) mountPoints() error {
	for _, mount := range app.Binds {
		err := app.mount(mount, false)
		if err != nil {
			return err
		}
	}
	for _, mount := range app.RoBinds {
		err := app.mount(mount, true)
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

	err = app.mountPoints()
	if err != nil {
		return err
	}

	// Invoke /proc/self/exe with 'child' subcommand.
	// `jailingo child` subcommand mounts /proc. Then, start target process.
	cmd := exec.Command("/proc/self/exe", append([]string{"child", "--log.level", app.logLevel, "--root", app.Root, "--", app.Command}, app.Args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
	}
	if err := cmd.Run(); err != nil {
		log.Fatal("ERROR: ", err)
		os.Exit(1)
	}

	return nil
}
