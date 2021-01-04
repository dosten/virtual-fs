
# Virtual Filesystem

## How to Use

```
$ go run cmd/virtual-fs
```

## Available Commands

```
TOUCH <name> - Creates a new file
MKDIR <name> - Creates a new directory
LS [-r] - Lists the files/directories inside the current working directory with support of recursive mode
PWD - Prints the current working directory
CD <path> - Changes the current working directory
QUIT - Quits the filesystem
```

## Example

```
> mkdir quz
> cd quz
> mkdir foo
> cd foo
> touch test
> pwd
/quz/foo/
> ls
TYPE    SIZE    NAME
file    0kb     test
> quit
```

