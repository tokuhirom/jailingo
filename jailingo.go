package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	core "github.com/tokuhirom/jailingo/core"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"os"
)

const VERSION = "0.0.1"

type stringArray []string

func (i *stringArray) String() string {
	return "stringArray"
}

func (i *stringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	app := kingpin.New("jailingo", "A command-line chat application.")
	levelString := app.Flag("log.level", "log level").Default("INFO").String()
	root := app.Flag("root", "chroot root").Required().String()
	binds := app.Flag("bind", "binds").Strings()
	preset := app.Flag("preset", "Load preset").Short('R').Default("true").Bool()

	run := app.Command("run", "Run command")
	runCommand := run.Arg("command", "Command").Required().String()
	runArgs := run.Arg("arguments", "Arguments").Strings()

	app.Command("unmount", "unmount")

	app.Command("version", "Show version and exit")

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

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
		robinds = []string{
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
		}
	}

	switch command {
	case "run":
		app := core.NewJailingApp(*root, tmpdirs, copyfiles, *binds, robinds, *runCommand, *runArgs)
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
	case "version":
		fmt.Printf("%v\n", VERSION)
		return
	}
}
