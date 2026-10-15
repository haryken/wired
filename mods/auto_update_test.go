package mods

import (
	"archive/tar"
	"bytes"
	"errors"
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
		t.Fatalf("same build: %+v", r)
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

func TestCheckOfficialOtaCaches(t *testing.T) {
	calls := 0
	withFakeOfficialOta(t, "3.0.1.888d", 404, nil, nil)
	officialOtaFetch = func(string) (int, int64, []byte, error) { calls++; return 404, 0, nil, nil }
	checkOfficialOta(false)
	checkOfficialOta(false)
	if calls != 1 {
		t.Fatalf("fetched %d times without force, want 1", calls)
	}
	checkOfficialOta(true)
	if calls != 2 {
		t.Fatalf("force did not refetch (calls=%d)", calls)
	}
}
