
Boogie is a replacement for FUNC, the Fedora Unified Network Controller.

This means it is a collection of programs for running arbitrary shell commands
securely in parallel across your network infrastructure.

It's early stages though and by no means production ready.

If you are interested in the design behind Boogie, be sure to look at https://github.com/dgryski/boogie/wiki/Design-considerations first.

Quick start:

Start bagent and boogied.

    ml36806:bin rbastic$ ./bcli -h "127.0.0.1:8081" 'ls'
    2014/06/09 13:38:18 req= {[127.0.0.1:8081] [ls] 10}
    2014/06/09 13:38:18 response:  {1402313898571918047}
    2014/06/09 13:38:21 host: localhost:8081
    stderr:
    stdout: bagent
    bcli
    boogied

See how easy it can be?

TODO: More complicated examples. ;)
