package fusefrontend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/rfjakob/gocryptfs/v2/internal/audit_log"
	"github.com/rfjakob/gocryptfs/v2/internal/nametransform"
	"github.com/rfjakob/gocryptfs/v2/internal/syscallcompat"
	"github.com/rfjakob/gocryptfs/v2/internal/tlog"
)

// Open - FUSE call. Open already-existing file.
//
// Symlink-safe through Openat().
func (n *Node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	dirfd, cName, errno := n.prepareAtSyscallMyself()
	if errno != 0 {
		return
	}

  // Test disallow: either in the disallowed prefix or called by cat
  path := n.GetFullFilepath()
  disallowed_path_prefix := "disallowed_to_read/"
  is_disallowed_prefix := strings.HasPrefix(path, disallowed_path_prefix)
  ctx2 := toFuseCtx(ctx)
  caller, _ := audit_log.GetCallerProcess(ctx2)
  disallowed_binary := "/usr/bin/cat"
  is_called_by_cat := caller == disallowed_binary
  tlog.Debug.Println(caller)
  tlog.Debug.Println("called")

  if (is_disallowed_prefix || is_called_by_cat) {
    m := make(map[string]string)
    if (is_disallowed_prefix) {
      tlog.Warn.Printf("prohibited because of prefix \"%s\"", disallowed_path_prefix)
      m["path"] = path
      m["matching_rule"] = disallowed_path_prefix
      audit_log.WriteAuditEvent(audit_log.EventProhibitedPathPrefix, ctx2, m)
    }  else { // is_called_by_cat
      m["disallowed_binary"] = disallowed_binary
      tlog.Warn.Printf("prohibited because called by \"%s\"", disallowed_binary)
      audit_log.WriteAuditEvent(audit_log.EventProhibitedCaller, ctx2, m)
    }
    errno = fs.ToErrno(os.ErrPermission)
    return
  }

	defer syscall.Close(dirfd)
	rn := n.rootNode()
	newFlags := rn.mangleOpenFlags(flags)
	// Taking this lock makes sure we don't race openWriteOnlyFile()
	rn.openWriteOnlyLock.RLock()
	defer rn.openWriteOnlyLock.RUnlock()


  // Playground
  // 1. get pid, uid, gid, pid_path
  {
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
  }
  // 2. get full filepath
  {
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
  }

	if rn.args.KernelCache {
		fuseFlags = fuse.FOPEN_KEEP_CACHE
	}

	// Open backing file
	fd, err := syscallcompat.Openat(dirfd, cName, newFlags, 0)


	// Handle a few specific errors
	if err != nil {
		if err == syscall.EMFILE {
			var lim syscall.Rlimit
			syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
			tlog.Warn.Printf("Open %q: too many open files. Current \"ulimit -n\": %d", cName, lim.Cur)
		}
		if err == syscall.EACCES && (int(flags)&syscall.O_ACCMODE) == syscall.O_WRONLY {
			fd, err = rn.openWriteOnlyFile(dirfd, cName, newFlags)
		}
	}
	// Could not handle the error? Bail out
	if err != nil {
		errno = fs.ToErrno(err)
		return
	}
  var f *File
	f, _, errno = NewFile(fd, cName, rn)
  fh = f
  ctx2 = toFuseCtx(ctx)
  m := n.GetAuditPayload(f, nil)
  audit_log.WriteAuditEvent(audit_log.EventOpen, ctx2, m)
	return fh, fuseFlags, errno
}

// Create - FUSE call. Creates a new file.
//
// Symlink-safe through the use of Openat().
func (n *Node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (inode *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	dirfd, cName, errno := n.prepareAtSyscall(name)
	if errno != 0 {
		return
	}
	defer syscall.Close(dirfd)

	var err error
	fd := -1
	// Make sure context is nil if we don't want to preserve the owner
	rn := n.rootNode()
	if !rn.args.PreserveOwner {
		ctx = nil
	}
	ctx2 := toFuseCtx(ctx)
	newFlags := rn.mangleOpenFlags(flags)
	// Handle long file name
	if !rn.args.PlaintextNames && nametransform.IsLongContent(cName) {
		// Create ".name"
		err = rn.nameTransform.WriteLongNameAt(dirfd, cName, name)
		if err != nil {
			return nil, nil, 0, fs.ToErrno(err)
		}
		// Create content
		fd, err = syscallcompat.OpenatUser(dirfd, cName, newFlags|syscall.O_CREAT|syscall.O_EXCL, mode, ctx2)
		if err != nil {
			nametransform.DeleteLongNameAt(dirfd, cName)
		}
	} else {
		// Create content, normal (short) file name
		fd, err = syscallcompat.OpenatUser(dirfd, cName, newFlags|syscall.O_CREAT|syscall.O_EXCL, mode, ctx2)
	}
	if err != nil {
		// xfstests generic/488 triggers this
		if err == syscall.EMFILE {
			var lim syscall.Rlimit
			syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
			tlog.Warn.Printf("Create %q: too many open files. Current \"ulimit -n\": %d", cName, lim.Cur)
		}
		return nil, nil, 0, fs.ToErrno(err)
	}

	f, st, errno := NewFile(fd, cName, rn)
  fh = f
  m := n.GetAuditPayload(f, &name)
  // the node information only contains the path *to* the file

  audit_log.WriteAuditEvent(audit_log.EventCreate, ctx2, m)
	if errno != 0 {
		return
	}

	inode = n.newChild(ctx, st, out)

	if rn.args.ForceOwner != nil {
		out.Owner = *rn.args.ForceOwner
	}

	return inode, fh, fuseFlags, errno
}
