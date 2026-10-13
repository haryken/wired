package vars

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PersistentRoot is shared by vic-engine, vic-cloud and vic-switchboard. They
// run as non-root anki-group users and create their own subdirectories here, so
// wired (root) must never leave a root:root 0755 directory in this tree. After a
// user-data clear, vic-switchboard (user net) exits with fault 913 when it
// cannot create persistent/switchboard.
const PersistentRoot = "/data/data/com.anki.victor/persistent"

const ankiFallbackID = 2901

// MkdirAll behaves like os.MkdirAll, but hands any directory it leaves under
// PersistentRoot back to anki:anki.
func MkdirAll(dir string) error {
	if err := os.MkdirAll(dir, 0775); err != nil {
		return err
	}
	FixPersistentOwnership(dir)
	return nil
}

// FixPersistentOwnership walks from dir up to PersistentRoot and repairs any
// root-owned directory. Directories already owned by a daemon are untouched.
func FixPersistentOwnership(dir string) {
	dir = filepath.Clean(dir)
	if dir != PersistentRoot && !strings.HasPrefix(dir, PersistentRoot+"/") {
		return
	}
	uid, gid := ankiIDs()
	for p := dir; ; p = filepath.Dir(p) {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid == 0 {
				mode := os.FileMode(0775)
				if p == PersistentRoot {
					mode = 0770
				}
				_ = os.Chown(p, uid, gid)
				_ = os.Chmod(p, mode)
			}
		}
		if p == PersistentRoot || p == "/" {
			return
		}
	}
}

// AnkiIDs returns the anki uid/gid (fallback 2901).
func AnkiIDs() (int, int) { return ankiIDs() }

func ankiIDs() (int, int) {
	uid, gid := ankiFallbackID, ankiFallbackID
	if u, err := user.Lookup("anki"); err == nil {
		if v, err := strconv.Atoi(u.Uid); err == nil {
			uid = v
		}
		if v, err := strconv.Atoi(u.Gid); err == nil {
			gid = v
		}
	}
	return uid, gid
}
