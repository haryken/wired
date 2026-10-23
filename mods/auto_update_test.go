package mods

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func otaHead(t *testing.T, manifest string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.ini", Mode: 0400, Size: int64(len(manifest))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "apq8009-robot-boot.img.gz", Mode: 0400, Size: 8192}); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if len(b) > otaHeadBytes {
		b = b[:otaHeadBytes]
	}
	return b
}

func TestOtaManifestVersion(t *testing.T) {
	head := otaHead(t, "[META]\nmanifest_version=1.0.0\nupdate_version=3.0.1.999d\nankidev=1\n")
	if got := otaManifestVersion(head); got != "3.0.1.999d" {
		t.Fatalf("version = %q", got)
	}
	if got := otaManifestVersion([]byte("not a tar")); got != "" {
		t.Fatalf("garbage gave %q", got)
	}
}

func TestCompareOSVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"3.0.1.999d", "3.0.1.888d", 1},
		{"3.0.1.1000d", "3.0.1.999d", 1},
		{"3.0.1.666d", "3.0.1.666d", 0},
		{"3.0.1.555d", "3.0.1.666d", -1},
		{"3.0.2", "3.0.1.999d", 1},
	}
	for _, c := range cases {
		if got := compareOSVersions(c.a, c.b); got != c.want {
			t.Errorf("compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func withFakeOfficialOta(t *testing.T, current string, status int, head []byte, err error) {
	t.Helper()
	oldFetch, oldVer := officialOtaFetch, currentAnkiVersionFn
	officialOtaFetch = func(string) (int, int64, []byte, error) { return status, 123456, head, err }
	currentAnkiVersionFn = func() string { return current }
	officialOtaCached = nil
	t.Cleanup(func() {
		officialOtaFetch, currentAnkiVersionFn = oldFetch, oldVer
		officialOtaCached = nil
	})
}

func TestCheckOfficialOta(t *testing.T) {
	newer := otaHead(t, "update_version=3.0.1.999d\n")

	withFakeOfficialOta(t, "3.0.1.888d", 206, newer, nil)
	if r := checkOfficialOta(true); !r.Exists || !r.Newer || r.Same || r.UpdateVersion != "3.0.1.999d" || r.Size != 123456 {
		t.Fatalf("newer build: %+v", r)
	}

	withFakeOfficialOta(t, "3.0.1.999d", 206, newer, nil)
	if r := checkOfficialOta(true); !r.Exists || r.Newer || !r.Same {
		t.Fatalf("same version must not be offered: %+v", r)
	}

	withFakeOfficialOta(t, "3.0.2.111d", 206, newer, nil)
	if r := checkOfficialOta(true); !r.Exists || !r.Newer || r.Same {
		t.Fatalf("older file on server must still be offered: %+v", r)
	}

	withFakeOfficialOta(t, "3.0.1.999d", 206, []byte("no manifest"), nil)
	if r := checkOfficialOta(true); !r.Exists || !r.Newer || r.Same {
		t.Fatalf("unreadable version: %+v", r)
	}

	withFakeOfficialOta(t, "3.0.1.999d", 404, nil, nil)
	if r := checkOfficialOta(true); !r.Reachable || r.Exists || r.Newer {
		t.Fatalf("missing file: %+v", r)
	}

	withFakeOfficialOta(t, "3.0.1.999d", 0, nil, errors.New("dial tcp: timeout"))
	if r := checkOfficialOta(true); r.Reachable || r.Exists || r.Error == "" {
		t.Fatalf("offline: %+v", r)
	}
}

func withUploadDir(t *testing.T, free int64) {
	t.Helper()
	oldDir, oldFree := otaUploadDir, otaFreeBytesFn
	otaUploadDir = filepath.Join(t.TempDir(), "ota-upload")
	otaFreeBytesFn = func(string) int64 { return free }
	t.Cleanup(func() { otaUploadDir, otaFreeBytesFn = oldDir, oldFree })
}

func fullOta(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	manifest := "[META]\nupdate_version=3.0.1.999d\n"
	for _, f := range []struct {
		name string
		data []byte
	}{
		{"manifest.ini", []byte(manifest)},
		{"apq8009-robot-boot.img.gz", bytes.Repeat([]byte{7}, 9000)},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0400, Size: int64(len(f.data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReceiveOtaUpload(t *testing.T) {
	withUploadDir(t, 1<<30)
	ota := fullOta(t)
	ver, err := receiveOtaUpload(bytes.NewReader(ota), int64(len(ota)))
	if err != nil || ver != "3.0.1.999d" {
		t.Fatalf("valid upload: ver=%q err=%v", ver, err)
	}
	got, err := os.ReadFile(otaUploadFile())
	if err != nil || !bytes.Equal(got, ota) {
		t.Fatalf("stored file differs (err=%v)", err)
	}
	if _, err := os.Stat(otaUploadFile() + ".part"); !os.IsNotExist(err) {
		t.Fatalf(".part left behind: %v", err)
	}
}

func TestReceiveOtaUploadRejects(t *testing.T) {
	ota := fullOta(t)
	garbage := bytes.Repeat([]byte("x"), 8192)
	cases := []struct {
		name string
		body []byte
		size int64
		free int64
	}{
		{"not an ota", garbage, int64(len(garbage)), 1 << 30},
		{"truncated", ota[:len(ota)/2], int64(len(ota)), 1 << 30},
		{"no space", ota, int64(len(ota)), 1 << 20},
		{"unknown size", ota, -1, 1 << 30},
		{"too large", ota, otaUploadMax + 1, 1 << 40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withUploadDir(t, c.free)
			if _, err := receiveOtaUpload(bytes.NewReader(c.body), c.size); err == nil {
				t.Fatal("expected an error")
			}
			if _, err := os.Stat(otaUploadDir); !os.IsNotExist(err) {
				t.Fatalf("upload dir left behind: %v", err)
			}
		})
	}
}

func TestCheckOfficialOtaFallbackLink(t *testing.T) {
	head := otaHead(t, "update_version=3.0.1.999d\n")
	backup := officialOtaURLs[1]
	cases := []struct {
		name    string
		primary func() (int, error)
		wantURL string
	}{
		{"primary missing", func() (int, error) { return 404, nil }, backup},
		{"primary unreachable", func() (int, error) { return 0, errors.New("timeout") }, backup},
		{"primary has file", func() (int, error) { return 206, nil }, officialOtaURLs[0]},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withFakeOfficialOta(t, "3.0.1.888d", 0, nil, nil)
			officialOtaFetch = func(u string) (int, int64, []byte, error) {
				if u == officialOtaURLs[0] {
					st, err := c.primary()
					if st == 206 {
						return st, 1, head, err
					}
					return st, 0, nil, err
				}
				return 206, 777, head, nil
			}
			r := checkOfficialOta(true)
			if !r.Exists || !r.Newer || r.URL != c.wantURL || r.Error != "" {
				t.Fatalf("%+v", r)
			}
		})
	}

	withFakeOfficialOta(t, "3.0.1.888d", 404, nil, nil)
	if r := checkOfficialOta(true); r.Exists || !r.Reachable || r.Error != "" {
		t.Fatalf("both missing: %+v", r)
	}
}

func TestCheckOfficialOtaCaches(t *testing.T) {
	calls := 0
	withFakeOfficialOta(t, "3.0.1.888d", 404, nil, nil)
	officialOtaFetch = func(string) (int, int64, []byte, error) { calls++; return 404, 0, nil, nil }
	n := len(officialOtaURLs)
	checkOfficialOta(false)
	checkOfficialOta(false)
	if calls != n {
		t.Fatalf("fetched %d times without force, want %d", calls, n)
	}
	checkOfficialOta(true)
	if calls != 2*n {
		t.Fatalf("force did not refetch (calls=%d)", calls)
	}
}
