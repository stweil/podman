//go:build !windows
// +build !windows

package graphdriver

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
)

type inode struct {
	Dev uint64
	Ino uint64
}

type platformChowner struct {
	mutex  sync.Mutex
	inodes map[inode]bool
}

func newLChowner() *platformChowner {
	return &platformChowner{
		inodes: make(map[inode]bool),
	}
}

func (c *platformChowner) LChown(path string, info os.FileInfo, toHost, toContainer *idtools.IDMappings) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	i := inode{
		Dev: uint64(st.Dev),
		Ino: uint64(st.Ino),
	}
	c.mutex.Lock()
	_, found := c.inodes[i]
	if !found {
		c.inodes[i] = true
	}
	c.mutex.Unlock()

	if found {
		return nil
	}

	// Map an on-disk UID/GID pair from host to container
	// using the first map, then back to the host using the
	// second map.  Skip that first step if they're 0, to
	// compensate for cases where a parent layer should
	// have had a mapped value, but didn't.
	uid, gid := int(st.Uid), int(st.Gid)
	if toContainer != nil {
		pair := idtools.IDPair{
			UID: uid,
			GID: gid,
		}
		mappedUID, mappedGID, err := toContainer.ToContainer(pair)
		if err != nil {
			if (uid != 0) || (gid != 0) {
				return fmt.Errorf("error mapping host ID pair %#v for %q to container: %v", pair, path, err)
			}
			mappedUID, mappedGID = uid, gid
		}
		uid, gid = mappedUID, mappedGID
	}
	if toHost != nil {
		pair := idtools.IDPair{
			UID: uid,
			GID: gid,
		}
		mappedPair, err := toHost.ToHost(pair)
		if err != nil {
			return fmt.Errorf("error mapping container ID pair %#v for %q to host: %v", pair, path, err)
		}
		uid, gid = mappedPair.UID, mappedPair.GID
	}
	if uid != int(st.Uid) || gid != int(st.Gid) {
		cap, err := system.Lgetxattr(path, "security.capability")
		if err != nil && !errors.Is(err, system.EOPNOTSUPP) && !errors.Is(err, system.EOVERFLOW) && err != system.ErrNotSupportedPlatform {
			return fmt.Errorf("%s: %v", os.Args[0], err)
		}

		// Make the change.
		if err := system.Lchown(path, uid, gid); err != nil {
			return fmt.Errorf("%s: %v", os.Args[0], err)
		}
		// Restore the SUID and SGID bits if they were originally set.
		if (info.Mode()&os.ModeSymlink == 0) && info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			if err := system.Chmod(path, info.Mode()); err != nil {
				return fmt.Errorf("%s: %v", os.Args[0], err)
			}
		}
		if cap != nil {
			if err := system.Lsetxattr(path, "security.capability", cap, 0); err != nil {
				return fmt.Errorf("%s: %v", os.Args[0], err)
			}
		}

	}
	return nil
}
