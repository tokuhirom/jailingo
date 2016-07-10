all: jailingo

jailingo:
	go build
	setcap CAP_SYS_ADMIN+ep jailingo
	setcap CAP_SYS_CHROOT+ep jailingo

test:
	go test

.PHONY: all

