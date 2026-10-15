package mods

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/os-vector/wired/vars"
)

// AI settings (keys, voices, identity) live under /data, which Clear User
// Data wipes. A copy on the separate persist partition is restored after a
// wipe so the user does not have to enter every key again.
//
// setup-persist.service is disabled in WireOS, so /persist is only a
// directory on the read-only rootfs. wired mounts the partition privately
// instead of over /persist, which other code still uses as a plain directory.
var (
	aiPersistDevice = "/dev/block/bootdevice/by-name/persist"
	aiPersistMount  = "/run/wireos-persist"
	aiBackupDir     = "/run/wireos-persist/wireos-ai-backup"
	// Written by mount-data when it formats /data after Clear User Data.
	aiDataClearedFlag = "/tmp/data_cleared"
	// Once per boot, so a wired restart never overwrites newer settings.
	aiRestoreDoneFlag = "/run/wireos-ai-restored"
)

const (
	aiBackupInterval    = 20 * time.Second
	aiPersistMountWait  = 30 * time.Second
	aiReloadSupportWait = 90 * time.Second
)

type aiBackupFile struct {
	dataPath func() string
	name     string
}

// vic-cloud (user cloud, group anki) reads and rewrites these files.
const aiRestoredMode os.FileMode = 0660

func aiBackupFiles() []aiBackupFile {
	return []aiBackupFile{
		{func() string { return xiaozhiConfigPath }, "xiaozhi.json"},
		{func() string { return openAIConfigPath }, "openai.json"},
	}
}

// StartAIConfigBackup restores the AI settings after a wipe, then keeps the
// backup in sync with every later change (wired or vic-cloud writes).
func StartAIConfigBackup() {
	go func() {
		if !mountPersist(aiPersistMountWait) {
			log.Println("[AIBackup] persist partition not mounted; backup disabled this boot")
			return
		}
		if restoreAIConfig() {
			go applyRestoredAIConfig()
		}
		for {
			backupAIConfig()
			time.Sleep(aiBackupInterval)
		}
	}()
}

func mountPersist(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if persistMounted() {
			return true
		}
		_ = os.MkdirAll(aiPersistMount, 0700)
		err := syscall.Mount(aiPersistDevice, aiPersistMount, "ext4",
			syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOATIME, "")
		if err == nil {
			return true
		}
		if time.Now().After(deadline) {
			log.Printf("[AIBackup] mount %s: %v", aiPersistDevice, err)
			return false
		}
		time.Sleep(2 * time.Second)
	}
}

func persistMounted() bool {
	var p, root syscall.Stat_t
	if syscall.Stat(aiPersistMount, &p) != nil || syscall.Stat(filepath.Dir(aiPersistMount), &root) != nil {
		return false
	}
	return p.Dev != root.Dev
}

// restoreAIConfig copies the backup back when /data was just wiped or a file
// is missing. Returns true when anything was restored.
func restoreAIConfig() bool {
	if _, err := os.Stat(aiRestoreDoneFlag); err == nil {
		return false
	}
	_, clearErr := os.Stat(aiDataClearedFlag)
	wiped := clearErr == nil
	restored := false
	for _, f := range aiBackupFiles() {
		backup, err := os.ReadFile(filepath.Join(aiBackupDir, f.name))
		if err != nil || !validAIConfig(backup) {
			continue
		}
		dst := f.dataPath()
		if _, err := os.Stat(dst); err == nil && !wiped {
			continue
		}
		if err := writeRestoredAIConfig(f, dst, backup); err != nil {
			log.Printf("[AIBackup] restore %s: %v", f.name, err)
			continue
		}
		log.Printf("[AIBackup] restored %s (wiped=%v)", f.name, wiped)
		restored = true
	}
	_ = os.WriteFile(aiRestoreDoneFlag, []byte("1"), 0644)
	return restored
}

func writeRestoredAIConfig(f aiBackupFile, dst string, data []byte) error {
	if err := vars.MkdirAll(filepath.Dir(dst)); err != nil {
		return err
	}
	tmp := dst + ".restore"
	if err := writeFileSync(tmp, data, aiRestoredMode); err != nil {
		return err
	}
	uid, gid := vars.AnkiIDs()
	_ = os.Chown(tmp, uid, gid)
	_ = os.Chmod(tmp, aiRestoredMode)
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	syncDir(filepath.Dir(dst))
	return nil
}

func writeFileSync(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// vic-cloud may already be running on defaults; wait until it can reload in
// place so the boot does not also restart vic-switchboard.
func applyRestoredAIConfig() {
	deadline := time.Now().Add(aiReloadSupportWait)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(xiaozhiReloadSupportedPath); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	applyVicCloudConfig()
}

func backupAIConfig() {
	if err := os.MkdirAll(aiBackupDir, 0700); err != nil {
		log.Printf("[AIBackup] mkdir: %v", err)
		return
	}
	for _, f := range aiBackupFiles() {
		data, err := os.ReadFile(f.dataPath())
		if err != nil || !validAIConfig(data) {
			continue
		}
		dst := filepath.Join(aiBackupDir, f.name)
		if old, err := os.ReadFile(dst); err == nil && bytes.Equal(old, data) {
			continue
		}
		tmp := dst + ".tmp"
		if err := writeFileSync(tmp, data, 0600); err != nil {
			log.Printf("[AIBackup] write %s: %v", f.name, err)
			continue
		}
		if err := os.Rename(tmp, dst); err != nil {
			log.Printf("[AIBackup] rename %s: %v", f.name, err)
			continue
		}
		syncDir(aiBackupDir)
	}
}

func validAIConfig(data []byte) bool {
	var v map[string]json.RawMessage
	return len(bytes.TrimSpace(data)) > 0 && json.Unmarshal(data, &v) == nil
}

func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}
