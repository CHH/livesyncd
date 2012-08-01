# livesyncd

`livesyncd` is a small One-Way Sync daemon, which operates in a similar
way to th "Automatic Deployment" feature, present in most modern IDEs.

## Get

Via `go get`:

	go get https://github.com/CHH/livesyncd

## Use

### Prerequisities on the Server Side

The server needs the OpenSSH daemon running. On most Linux distributions
installing the `openssh-server` or `opensshd` packages as well as
starting the service will do.

Your computer of course needs access to the server via SSH, of course.
I'm recommending adding your Public Key to the `~/.ssh/authorized_keys`
of the user you want to use for accessing the server.

### Starting to sync

`livesyncd` monitors a single directory tree for changes and tries to
mirror the directory structure on the remote side.

To use `livesyncd` you've to tell it at least the server (`--remote-host`)
and the root directory for mirroring on the server (`--remote-root`).

	% livesyncd --remote-host user@myserver --remote-root /tmp

Now try this in the working directory of `livesyncd`:

	echo "foo" > foo.txt

Then wait a moment (typically under a second) and try:

	% ssh user@myserver 'cat /tmp/foo.txt'

You should get a single `foo` as output from the server. This means the
file is uploaded!

Now delete the local file:

	% rm foo.txt

Try to cat the file again:

	% ssh user@myserver 'cat /tmp/foo.txt'

You now should get:

	cat: /tmp/foo.txt: No such file or directory

_Hurray!_

## Common Problems

### Users of IntelliJ IDEA

**TL;DR:** Turn off "Save Write" in "Preferences > General".

IntelliJ IDEA includes a so called "Save Write" setting, which you can
find in "Preferences -> General". When "Save Write" is turned on, then
each time a file is saved, the new contents are written to a temporary
file, then the old file is renamed to a temporary name and then the file
containing the saved contents is renamed to the real file name.

I'm not far enough to detect this series of events, and so it can't
mirror such changes on the remote host. So for using `livesyncd`, you've
to turn off "Save Write".

## License

livesyncd is licensed under the terms of the MIT License, which is
bundled in the file `LICENSE`.

