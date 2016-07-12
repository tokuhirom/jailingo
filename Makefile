all: jailingo

jailingo: jailingo.go core/unmount.go core/app.go child/child.go Makefile
	go build
	sudo setcap CAP_SYS_ADMIN,CAP_SYS_CHROOT,CAP_MKNOD+ep-i jailingo

test:
	go test

clean:
	rm jailingo

.PHONY: all clean

