bboltbrowser
===========

Fork from https://github.com/br0xen/boltbrowser but using bbolt instead

Installing
----------

Install in the standard way:

```sh
go get github.com/eduardonunesp/bboltbrowser
```

Usage
-----

Just provide a bboltDB filename to be opened as the first argument on the command line:

```bash
bboltbrowser <filename>
```

To see all options that are available, run:

```bash
bboltbrowser --help
```

Troubleshooting
---------------

If you're having trouble with garbled characters being displayed on your screen, you may try a different value for `TERM`.
People tend to have the best luck with `xterm-256color` or something like that. Play around with it and see if it fixes your problems.
