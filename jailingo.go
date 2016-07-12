package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	core "github.com/tokuhirom/jailingo/core"
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
	root := flag.String("root", "", "chroot root")
	levelString := flag.String("log.level", "INFO", "log level")
	preset := flag.Bool("R", true, "Load preset")
	var binds stringArray
	flag.Var(&binds, "bind", "binds")
	version := flag.Bool("version", false, "Show version and exit")
	unmount := flag.Bool("unmount", false, "Do unmount and exit")
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

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s: %s [OPTIONS...] -- /path/to/executable --arg1 arg2\n\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
		return
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

	app := core.NewJailingApp(*root, tmpdirs, copyfiles, binds, robinds, flag.Args())
	if *unmount {
		err = app.UnmountAll()
		if err != nil {
			log.Fatal("Cannot unmount: ", err)
		}
	} else {
		err = app.Main()
		if err != nil {
			log.Fatal("Cannot run jailingo: ", err)
		}
	}
}
