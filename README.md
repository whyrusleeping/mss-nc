# multistream netcat

A basic netcat like tool for communicating with multistream protocol servers.

usage:
```bash
$ mss-nc <host> <port>
```

Once youre connected, you can either list available protocols with `ls` or you
can select a protocol by typing its name. Once a protocol is selected, the input
mode changes from varint prefixed and newline delimited to raw.
