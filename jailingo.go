package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	core "github.com/tokuhirom/jailingo/core"
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

	app := core.NewJailingApp(*root, binds)
	err = app.Main()
	if err != nil {
		log.Fatal("Cannot run jailingo: ", err)
	}
}
