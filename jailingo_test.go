package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	r, w, _ := os.Pipe()
	cmd := exec.Command("./jailingo", "--version")
	cmd.Stderr = w

	err := cmd.Run()
	if err != nil {
		t.Error(err)
	}

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	if buf.String() != "0.0.1\n" {
		t.Error("Failure: " + buf.String())
	}
}

func TestRun(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Error(err)
	}
	t.Log(tmpdir)

	r, w, _ := os.Pipe()
	cmd := exec.Command("./jailingo", "run", "--root", tmpdir, "/bin/ls")
	cmd.Stdout = w

	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	if !strings.Contains(buf.String(), "bin") {
		t.Error("Failure: " + buf.String())
	}
}
