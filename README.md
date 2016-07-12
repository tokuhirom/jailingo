Dependencies

    Linux supports capability

Usage

    go build
    sudo setcap CAP_SYS_ADMIN,CAP_SYS_CHROOT,CAP_MKNOD+ep-i jailingo

WARNINGS

Do not run this command with 'sudo'. This command won't drop capabilities unlike [jailing](https://github.com/kazuho/jailing/blob/master/jailing)
Because, jailingo uses setcap flag. It's not inherit to child process.

