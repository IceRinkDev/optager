# optager

optager is a command-line tool for installing programs that come packaged in
e.g. .tar.gz format into the /opt/ folder. It takes away all the hassle of
extracting the archive and then linking the binaries into the right places.
optager is (or at least tries to be) compliant with the Filesystem Hierarchy
Standard of Linux as well as the XDG Base Directory Specification.

## Installation

To install optager on your system you have to have `git` and `go` installed.
You can then clone this repository, change into the cloned directory and then
run

```sh
go install
```

In case the `optager` command is not found you might have to add `$HOME/go/bin`
to your PATH. This can be done by adding the following line to your .bashrc or
.zshrc.

```sh
export PATH="$PATH:$HOME/go/bin"
```

## Usage

For detailed information on the commands and available flags run the command in
question with `-h` or `--help` added.

### Install a package

This will install the package into the /opt/ folder and print all new binaries
that are available to you.

```sh
optager install <path-to-the-archive>
```

### List all installed packages

Optager can list all packages that were installed using `optager install`.

```sh
optager list
```

### Remove a package

This will delete the package folder in /opt/ and all symbolic links created by
optager.

```sh
optager remove <package-name>
```

or alternatively

```sh
optager uninstall <package-name>
```
