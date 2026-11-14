package mods

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDnsMode(t *testing.T) {
	cases := map[string]string{
		"":         dnsModeDHCP,
		"dhcp":     dnsModeDHCP,
		"DHCP":     dnsModeDHCP,
		"public":   dnsModePublic,
		"PUBLIC":   dnsModePublic,
		"garbage":  dnsModeDHCP,
		"  public ": dnsModePublic,
	}
	for in, want := range cases {
		if got := normalizeDnsMode(in); got != want {
			t.Fatalf("normalizeDnsMode(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDnsHasPublic(t *testing.T) {
	if dnsHasPublic(nil) {
		t.Fatal("nil should be false")
	}
	if dnsHasPublic([]string{"192.168.1.1"}) {
		t.Fatal("router-only should be false")
	}
	if !dnsHasPublic([]string{dnsPublicPrimary, dnsPublicSecondary}) {
		t.Fatal("public pair should be true")
	}
	if !dnsHasPublic([]string{"1.1.1.1", dnsPublicPrimary}) {
		t.Fatal("primary anywhere should be true")
	}
}

func TestDnsSettingsHTTPRoundTrip(t *testing.T) {
	dir := t.TempDir()
	old := dnsConfigPath
	dnsConfigPath = filepath.Join(dir, "dns.json")
	t.Cleanup(func() { dnsConfigPath = old })

	m := NewDnsSettings()

	get := httptest.NewRequest("GET", "/api/mods/DnsSettings/get", nil)
	gw := httptest.NewRecorder()
	m.HTTP(gw, get)
	if gw.Code != 200 {
		t.Fatalf("get status %d", gw.Code)
	}
	var view map[string]interface{}
	if err := json.Unmarshal(gw.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["mode"] != dnsModeDHCP {
		t.Fatalf("default mode=%v", view["mode"])
	}

	form := url.Values{"mode": {"public"}}
	set := httptest.NewRequest("POST", "/api/mods/DnsSettings/set?"+form.Encode(), nil)
	set.Form = form
	sw := httptest.NewRecorder()
	m.HTTP(sw, set)
	if sw.Code != 200 {
		t.Fatalf("set status %d body=%s", sw.Code, sw.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(sw.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["status"] != "success" {
		t.Fatalf("set response=%v", out)
	}
	cfg := loadDnsCfg()
	if cfg.Mode != dnsModePublic {
		t.Fatalf("saved mode=%q", cfg.Mode)
	}
	raw, err := os.ReadFile(dnsConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", raw)
	}

	form2 := url.Values{"mode": {"dhcp"}}
	set2 := httptest.NewRequest("POST", "/api/mods/DnsSettings/set?"+form2.Encode(), nil)
	set2.Form = form2
	sw2 := httptest.NewRecorder()
	m.HTTP(sw2, set2)
	if loadDnsCfg().Mode != dnsModeDHCP {
		t.Fatalf("restore dhcp failed: %s", sw2.Body.String())
	}

	bad := httptest.NewRequest("POST", "/api/mods/DnsSettings/set?mode=evil", nil)
	bad.Form = url.Values{"mode": {"evil"}}
	bw := httptest.NewRecorder()
	m.HTTP(bw, bad)
	if bw.Code == http.StatusOK && jsonLooksSuccess(bw.Body.Bytes()) {
		t.Fatalf("evil mode accepted: %s", bw.Body.String())
	}
}

func jsonLooksSuccess(b []byte) bool {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return false
	}
	return m["status"] == "success"
}
