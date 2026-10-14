package mods

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func intp(n int) *int { return &n }

func TestXzCoursePools(t *testing.T) {
	seen := map[string]string{}
	for _, c := range xzCourses {
		if len(c.Books) == 0 {
			t.Fatalf("course %d has no books", c.Idx)
		}
		for v := 0; v < 2; v++ {
			pool := c.Voices[v].Pool
			if len(pool) != 10 {
				t.Fatalf("course %d voice %d pool size %d", c.Idx, v, len(pool))
			}
			for _, m := range pool {
				if prev, dup := seen[m]; dup {
					t.Fatalf("MAC %s in two pools (%s)", m, prev)
				}
				seen[m] = c.ID
				if macInXiaozhiViPool(m) {
					t.Fatalf("course MAC %s overlaps the vi pool", m)
				}
			}
			mac := xzPickCourseMAC(c.Idx, v, pool[0])
			if mac == pool[0] || !xzMACInCoursePool(c.Idx, v, mac) {
				t.Fatalf("pick course %d voice %d: %q", c.Idx, v, mac)
			}
		}
	}
}

func TestApplyXiaozhiCourseUpdateIdentity(t *testing.T) {
	cfg := XiaozhiCfg{Enabled: true, IdentityMode: xiaozhiIdentityViPool, DeviceID: "aa:bb:cc:dd:ee:ff"}

	if !applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Idx: intp(4)}) {
		t.Fatal("switching to course must change identity")
	}
	if cfg.IdentityMode != xiaozhiIdentityCourse || cfg.CourseIdx != 4 || !xzMACInCoursePool(4, 0, cfg.DeviceID) {
		t.Fatalf("bad course identity: %+v", cfg)
	}
	if cfg.ClientID == "" || cfg.Token != "" || cfg.OTABaseURL != xiaozhiTenclassOTA {
		t.Fatalf("bad connection fields: %+v", cfg)
	}

	mac := cfg.DeviceID
	name := " Linh "
	if applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Unit: intp(3), StudentName: &name}) {
		t.Fatal("unit/name change must not change identity")
	}
	if cfg.DeviceID != mac || cfg.StudentName != "Linh" || cfg.CourseUnit["4.0"] != 3 {
		t.Fatalf("lesson not stored: %+v", cfg)
	}

	if !applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Voice: intp(1)}) {
		t.Fatal("voice change must change identity")
	}
	if !xzMACInCoursePool(4, 1, cfg.DeviceID) {
		t.Fatalf("MAC %s not in voice B pool", cfg.DeviceID)
	}

	// Out-of-range unit/book are clamped; unknown course is ignored.
	if applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Idx: intp(99), Book: intp(7), Unit: intp(42)}) {
		t.Fatal("invalid values must not change identity")
	}
	if cfg.CourseIdx != 4 || cfg.CourseBook["4"] != 0 || cfg.CourseUnit["4.0"] != 1 {
		t.Fatalf("not clamped: %+v", cfg)
	}

	// Per-course lesson memory: back to Explorers keeps its own book/unit.
	applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Idx: intp(1), Book: intp(1), Unit: intp(5)})
	applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Idx: intp(4)})
	applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{Idx: intp(1)})
	if _, _, book, unit := xzCourseLesson(cfg); book != 1 || unit != 5 {
		t.Fatalf("explorers lesson lost: book %d unit %d", book, unit)
	}

	cfg.Enabled = false
	if !applyXiaozhiCourseUpdate(&cfg, xzCourseUpdate{}) || !cfg.Enabled {
		t.Fatal("set_course must enable Xiaozhi")
	}
}

func TestApplyVicCloudConfig(t *testing.T) {
	dir := t.TempDir()
	oldFlag, oldSup, oldFn := xiaozhiReloadFlagPath, xiaozhiReloadSupportedPath, restartVicCloudFn
	defer func() {
		xiaozhiReloadFlagPath, xiaozhiReloadSupportedPath, restartVicCloudFn = oldFlag, oldSup, oldFn
	}()
	xiaozhiReloadFlagPath = filepath.Join(dir, "run", "xiaozhi-reload")
	xiaozhiReloadSupportedPath = filepath.Join(dir, "supported")
	restarts := make(chan struct{}, 4)
	restartVicCloudFn = func() { restarts <- struct{}{} }

	// Old vic-cloud: debounced restart, three quick calls → one restart.
	for i := 0; i < 3; i++ {
		if applyVicCloudConfig() {
			t.Fatal("applied without support file")
		}
	}
	select {
	case <-restarts:
	case <-time.After(vicCloudRestartDebounce + 2*time.Second):
		t.Fatal("no restart")
	}
	select {
	case <-restarts:
		t.Fatal("restart not debounced")
	case <-time.After(300 * time.Millisecond):
	}
	if _, err := os.Stat(xiaozhiReloadFlagPath); err == nil {
		t.Fatal("flag written without support")
	}

	// New vic-cloud: flag file, no restart.
	if err := os.WriteFile(xiaozhiReloadSupportedPath, []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	if !applyVicCloudConfig() {
		t.Fatal("not applied in place")
	}
	if _, err := os.Stat(xiaozhiReloadFlagPath); err != nil {
		t.Fatalf("flag missing: %v", err)
	}
	select {
	case <-restarts:
		t.Fatal("restarted although supported")
	case <-time.After(vicCloudRestartDebounce + 500*time.Millisecond):
	}
}

func TestXiaozhiConnectionChanged(t *testing.T) {
	a := XiaozhiCfg{Enabled: true, DeviceID: "x", IdentityMode: "vi_pool", ListenMode: "fixed"}
	b := a
	b.ListenMode = "vad"
	b.IdentityMode = "VI_POOL"
	if xiaozhiConnectionChanged(a, b) {
		t.Fatal("listen mode / case must not count")
	}
	b.DeviceID = "y"
	if !xiaozhiConnectionChanged(a, b) {
		t.Fatal("device id change missed")
	}
}

func TestApplyXiaozhiViPoolRenewsIdentity(t *testing.T) {
	cfg := XiaozhiCfg{DeviceID: xiaozhiViPoolMACs[0], ClientID: "old-client", Token: "tok"}
	applyXiaozhiViPool(&cfg)
	if cfg.DeviceID == xiaozhiViPoolMACs[0] || !macInXiaozhiViPool(cfg.DeviceID) {
		t.Fatalf("MAC = %q, want another pool MAC", cfg.DeviceID)
	}
	if cfg.ClientID == "" || cfg.ClientID == "old-client" {
		t.Fatalf("client ID = %q, want a fresh one", cfg.ClientID)
	}
	if cfg.Token != "" {
		t.Fatalf("token = %q, want cleared", cfg.Token)
	}
}
