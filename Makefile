all: jailingo

jailingo: jailingo.go
	go build
	sudo setcap CAP_SYS_ADMIN+ep jailingo
	sudo setcap CAP_SYS_CHROOT+ep jailingo

test:
	go test

clean:
	rm jailingo

.PHONY: all clean

