package main

import (
	log "github.com/Sirupsen/logrus"
	child "github.com/tokuhirom/jailingo/child"
	core "github.com/tokuhirom/jailingo/core"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"os"
	"path/filepath"
)

const VERSION = "0.0.1"

func filter(s []string, fn func(string) bool) []string {
	var p []string // == nil
	for _, v := range s {
		if fn(v) {
			p = append(p, v)
		}
	}
	return p
}

func Run(args []string) {
	app := kingpin.New("jailingo", "A command-line chat application.")
	app.Version(VERSION)
	levelString := app.Flag("log.level", "log level").Default("INFO").String()
	root := app.Flag("root", "chroot root").Required().String()
	binds := app.Flag("bind", "binds").Strings()
	preset := app.Flag("preset", "Load preset").Short('R').Default("true").Bool()

	run := app.Command("run", "Run command")
	runCommand := run.Arg("command", "Command").Required().String()
	runArgs := run.Arg("arguments", "Arguments").Strings()

	childProcSubCommand := app.Command("child", "Internal use only")
	childProcCommand := childProcSubCommand.Arg("command", "Command").String()
	childProcArgs := childProcSubCommand.Arg("arguments", "Arguments").Strings()

	app.Command("unmount", "unmount")

	command := kingpin.MustParse(app.Parse(args))

	level, err := log.ParseLevel(*levelString)
	if err != nil {
		log.Fatal("Invalid logging level: ", err)
	}
	log.SetLevel(level)

	if (*root)[0] != '/' {
		log.Fatal("--root argument must be absolute")
	}

	tmpdirs := []string{}
	copyfiles := []string{}
	robinds := []string{}
	if *preset {
		tmpdirs = []string{"/tmp", "/run/lock", "/var/tmp"}
		copyfiles = []string{
			"/etc/group",
			"/etc/passwd",
			"/etc/resolv.conf",
			"/etc/hosts",
		}
		robinds = filter([]string{
			"/bin",
			"/etc/alternatives",
			"/etc/pki/tls/certs",
			"/etc/pki/ca-trust",
			"/etc/ssl/certs",
			"/lib",
			"/lib64",
			"/sbin",
			"/usr/bin",
			"/usr/include",
			"/usr/lib",
			"/usr/lib64",
			"/usr/libexec",
			"/usr/sbin",
			"/usr/share",
			"/usr/src",
			"/xxy",
		}, func(path string) bool {
			if _, err := os.Stat(filepath.Join(path)); os.IsNotExist(err) {
				log.Debugf("Missing %v. Skip.", path)
				return false
			}
			return true
		})
		log.Infof("%v", robinds)
	}

	switch command {
	case "run":
		app := core.NewJailingApp(*root, tmpdirs, copyfiles, *binds, robinds, *runCommand, *runArgs, *levelString)
		err = app.Main()
		if err != nil {
			log.Fatal("Cannot run jailingo: ", err)
		}
	case "unmount":
		app := core.NewUnmounter(*root, *binds, robinds)
		err = app.UnmountAll()
		if err != nil {
			log.Fatal("Cannot unmount: ", err)
		}
	case "child":
		child := child.NewChildProcfs(*root, *childProcCommand, *childProcArgs, *levelString)
		err = child.Run()
		if err != nil {
			log.Fatal("Cannot unmount: ", err)
		}
	}
}

func main() {
	Run(os.Args[1:])
}
