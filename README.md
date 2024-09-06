# getnew

Looks in a directory for the nth newest file (not directories) and moves it to the current directory. Default is the top newest.

Usage:

```
getnew is a CLI tool that looks in a specified directory for the nth newest file
and moves it to the current directory. By default, it moves the newest file.

The source directory can be set using the GETNEW_SOURCE_DIR environment variable
or specified using the --source flag.

Optionally, provide a filter argument to match files partially.

Usage:
  getnew [filter] [flags]

Flags:
  -h, --help            help for getnew
  -n, --nth int         Nth newest file to move (default is 1, the newest) (default 1)
  -s, --source string   Source directory (overrides GETNEW_SOURCE_DIR)
```