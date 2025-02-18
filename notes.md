# Notes

Get started

```
umount plain
rm -rf cipher plain
mkdir cipher plain
./gocryptfs -init cipher
./gocryptfs cipher plain
```

Relevant params

```
--debug                                 Enable debug output
--fg                                    Stay in the foreground
--trace string                          Write execution trace to file
  - for gc and stuff, not for auditlog
--ro                                    Mount the filesystem read-only
--rw                                    Mount the filesystem read-write
```

```
./gocryptfs --debug --fg --fusedebug cipher plain
```

## Things I noted
- Each folder has a `gocryptfs.diriv` initialization vector

## Gocryptfs.conf
```json
{
	"Creator": "gocryptfs [no_tags_found]",
	"EncryptedKey": "...",
	"ScryptObject": {
		"Salt": "...",
		"N": 65536,
		"R": 8,
		"P": 1,
		"KeyLen": 32
	},
	"Version": 2,
	"FeatureFlags": [
		"HKDF",
		"GCMIV128",
		"DirIV",
		"EMENames",
		"LongNames",
		"Raw64"
	]
}
```

## Code structure
- [ ] `main`
  - [x] `cli_args`, `help`, `version`
    - CLI interaction
  - [x] `daemonize`
    - `forkChild` gets called iff not `-fg` and do mount.
      - Go doesnt have real fork, this lets the parent wait for the child to die
    - Also handles syslog output
  - [x] `fsck`
    - Only gets called from main when `-fsck`, terminates afterwards
    - "Check CIPHERDIR for consistency. If corruption is found, the exit code is 26."
  - [x] `info`
    - `-info`: Pretty prints config without secrets
  - [ ] `init_dir`
    - `gocryptfs -init` (for both fwd and bwd mode)
  - [ ] `main`
  - [x] `masterkey`
    - gets masterkey from args (which is hexified when given to user)
    - if via stdin, read password
    - if via param, just unhex it
    - if "zero-key" => all zeros (testing only)
  - [ ] `mount`
  - [x] `profiling`: Perf stuff
    - `-cpuprofile`: `pprof.StartCPUProfile`
    - `-memprofile`: `pprof.WriteHeapProfile`
    - `-trace`: `runtime/trace`

- ctlsock
  - Library for gocryptfs control socket interface
  - `-ctlsock /tmp/my.sock`
  - Used by `gocryptfs-xray`
- gocryptfs-xray
  - do encryption/decryption without the mounty stuff using the sockets
  - See man page

- [x] `internal::configfile`: load config file + key wrapping
  - Manages **master key storage**
  - Contains scrypt
  - how key wrapping works
    - random base master key is generated
    - another key is derived user password (using scrypt)
      - This is the key encryption key (KEK)
      - scrypt "allgedly increases entropy?" but more important slows down brute force
    - master key is encrypted/wrapped using derived key
      - Using AES-256-GCM, see `getKeyEncrypter`
    - encrypted master key is stored in cfg file
- [ ] `internal::contentenc`
- [ ] `internal::cryptocore`
- [ ] `internal::ctlsocksrv`
- [ ] `internal::ensurefds012`
- [x] `internal::exitcodes`: what name implies
- [x] `internal::fido2`: Impls fido2 support by wrapping `fido2-assert` and `fido2-cred`
  - can be spefidied using `--fido2` when `--init`
- [ ] `internal::fusefrontend`
- [x] `internal::fusefrontend_reverse`
  - we dont care about reverse mode, just implements same method heads as normal
- [ ] `internal::inomap`
- [ ] `internal::nametransform`
- [ ] `internal::openfiletable`
- [ ] `internal::pathiv`
- [ ] `internal::readpassword`
- [ ] `internal::siv_aead`
- [x] `internal::speed`: Run crypto speed test via `-speed`
  - Just one public runction that gets called only from main
- [ ] `internal::stupidgcm`
- [ ] `internal::syscallcompat`
- [x] `internal::tlog`: Basic logging library that can be toggled

### Folders to be ignored
- .git
- .github
- contrib
  - Just used for tests
- Documentation
- profiling
- tests

## FUSE funcs
```
~/code/gocryptfs$ rg -i "FUSE call" internal/fusefrontend
internal/fusefrontend/node_open_create.go
15:// Open - FUSE call. Open already-existing file.
57:// Create - FUSE call. Creates a new file.

internal/fusefrontend/node_dir_ops.go
70:// Mkdir - FUSE call. Create a directory at "newPath" with permissions "mode".
162:// Readdir - FUSE call.
247:// Rmdir - FUSE call.
379:// Opendir is a FUSE call to check if the directory can be opened.

internal/fusefrontend/node.go
23:// Lookup - FUSE call for discovering a file.
68:// GetAttr - FUSE call for stat()ing a file.
113:// Unlink - FUSE call. Delete a file.
138:// Readlink - FUSE call.
151:// Setattr - FUSE call. Called for chmod, truncate, utimens, ...
233:// StatFs - FUSE call. Returns information about the filesystem.
247:// Mknod - FUSE call. Create a device file.
301:// Link - FUSE call. Creates a hard link at "newPath" pointing to file
352:// Symlink - FUSE call. Create a symlink.
419:// Rename - FUSE call.

internal/fusefrontend/node_xattr.go
33:// GetXAttr - FUSE call. Reads the value of extended attribute "attr".
82:// SetXAttr - FUSE call. Set extended attribute.
107:// RemoveXAttr - FUSE call.
125:// ListXAttr - FUSE call. Lists extended attributes on the file at "relPath".

internal/fusefrontend/file.go
232:// Read - FUSE call
358:// Write - FUSE call
394:// Release - FUSE call, close file
407:// Flush - FUSE call
427:// Getattr FUSE call (like stat)

internal/fusefrontend/file_holes.go
60:// Lseek - FUSE call.

internal/fusefrontend/file_allocate_truncate.go
27:// Allocate - FUSE call for fallocate(2)
```

## Stacktrace of Read FUSE call
```
runtime/debug.Stack()
        runtime/debug/stack.go:26 +0x5e
runtime/debug.PrintStack()
        runtime/debug/stack.go:18 +0x13
github.com/rfjakob/gocryptfs/v2/internal/fusefrontend.(*File).Read(0xc000154070, {0x563e4beb5c40?, 0x1?}, {0xc000918000, 0x1000, 0x1000}, 0x0)
        github.com/rfjakob/gocryptfs/v2/internal/fusefrontend/file.go:235 +0x65
github.com/hanwen/go-fuse/v2/fs.(*rawBridge).Read(0x0?, 0xc000110230, 0xc0001c43e0, {0xc000918000, 0x1000, 0x1000})
        github.com/hanwen/go-fuse/v2@v2.5.0/fs/bridge.go:774 +0x169
github.com/hanwen/go-fuse/v2/fuse.doRead(0xc000128000, 0xc0001c4248)
        github.com/hanwen/go-fuse/v2@v2.5.0/fuse/opcode.go:398 +0x79
github.com/hanwen/go-fuse/v2/fuse.(*Server).handleRequest(0xc000128000, 0xc0001c4248)
        github.com/hanwen/go-fuse/v2@v2.5.0/fuse/server.go:527 +0x2d1
github.com/hanwen/go-fuse/v2/fuse.(*Server).loop(0xc000128000, 0x1)
        github.com/hanwen/go-fuse/v2@v2.5.0/fuse/server.go:500 +0x110
created by github.com/hanwen/go-fuse/v2/fuse.(*Server).readRequest in goroutine 5
        github.com/hanwen/go-fuse/v2@v2.5.0/fuse/server.go:367 +0x53e
```

## Find out how the FUSE funcs get set (equiv to C `fuse_operations`)
1. main.go: `func main():`
```
nOps := countOpFlags(&args)
if nOps == 0 {
	// Default operation: mount.
	doMount(&args)
	return // Don't call os.Exit to give deferred functions a chance to run
}
```

2. mount.go: `doMount(args *argContainer)`
```
// Initialize gocryptfs (read config file, ask for password, ...)
fs, wipeKeys := initFuseFrontend(args)
// Try to wipe secret keys from memory after unmount
defer wipeKeys()
// Initialize go-fuse FUSE server
srv := initGoFuse(fs, args)
if x, ok := fs.(AfterUnmounter); ok {
	defer x.AfterUnmount()
}
```

3b. mount.go: `initGoFuse(...)`
```
srv, err := fs.Mount(args.mountpoint, rootNode, fuseOpts)
```

- go interfaces use vtables
- thus it can "upcast" it to `fs.InodeEmbedder` and still resove the methods
- initGoFuse actually creates `fs` as a `RootNode` (`./internal/fusefrontend/root_node.go`)
- `RootNode` extends `Node`, which newtypes `fs.Inode`
  - This implements some FUSE calls (`Lookup`, `GetAttr`, `Unlink`, `Readlink`...)
- And in `Create`/`Open` (`node_open_create.go`) it calls `NewFile`, which returns a File


## FUSE Op Trail (Order)
- For Read OP
  - (Node) `GETATTTR`
  - `LOOKUP`
  - `OPEN`
  - `READ`
  - `FLUSH`
  - `RELEASE`
- For Write OP (new file)
  - `LOOKUP`
  - `CREATE`
  - `FLUSH`
  - `GETXATTR`
  - `WRITE`
  - `FLUSH`
  - `RELEASE`

## All FUSE Methods implemented
### Node
- Access (undocumented, never called by gocryptfs)
- Create
- Fsync
- Getattr
- GetXAttr
- Link
- ListXAttr
- Lookup
- Mkdir
- Mknod
- Open
- Opendir (if the directory can be opened)
- Readdir
- Readlink
- RemoveXAttr
- Rename
- Rmdir
- Setattr
- SetXAttr
- StatFs
- Symlink
- Unlink
### File
- Allocate
- Flush
- Getattr
- Lseek
- Read
- Release
- Write
- Setattr (undocumented)

## What to track for basic Audit trail
- Open file
  - Open
  - Create
- Close file
  - Release
- Read file
  - Read
  - Readlink
- Write file
  - Write
  - Rename
  - Unlink
- Other to not be confused
  - Lseek
  - Allocate (for fallocate)
  - Mkdir
  - Rmdir
  - MkNod (create a device file?!? lets hope nobody ever uses this...)
  - Link (hardlink?!?)
  - Symlink

## Tricks in codebase

### Get filepath from Node `n`
```go
var parts []string
var curr *fs.Inode
curr = &n.Inode
// traverse up
for curr != nil {
  name, parent := curr.Parent()
  parts = append(parts, name)
  curr = parent
  println(parts[0])
}
slices.Reverse(parts)
file_path := filepath.Join(parts...)
tlog.Debug.Printf("Called on: %s", file_path)
```

### Get identifiers for Node and File
(Currently not happy with it, but hte best I've got so far)
```go
f.intFd()
n.StableAttr().Ino
```

### Get caller pid, process name, uid, gid
Let `ctx` be a `context.Context`

```go
ctx2 := toFuseCtx(ctx)
pid := ctx2.Pid
uid := ctx2.Uid
gid := ctx2.Gid
buf := make([]byte, syscallcompat.PATH_MAX)
pid_path := fmt.Sprintf("/proc/%d/exe", pid)
num, err := syscall.Readlink(pid_path, buf)
if (err != nil) {
  tlog.Warn.Printf("read process name failed w/ '%s'", err)
} else {
  caller_str := string(buf[:num])
  tlog.Debug.Printf("pid %d uid %d gid %d process_name '%s'", pid, uid, gid, caller_str)
}
```
