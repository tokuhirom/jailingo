# Synopsis

    jailingo run -R --root /tmp/jail -- /usr/bin/ansible-playbook main.yml

# Dependencies

    Linux supports capability

# Build

    make

# WARNINGS

Do not run this command with 'sudo'. This command won't drop capabilities unlike [jailing](https://github.com/kazuho/jailing/blob/master/jailing)
Because, jailingo uses setcap flag. It's not inherit to child process.

