package mods

import "testing"

func TestSsidFromConnmanPathHuynh(t *testing.T) {
	got := ssidFromConnmanPath("wifi_000af51c1ea0_4875796e6820322e34_managed_psk")
	if got != "Huynh 2.4" {
		t.Fatalf("Huynh 2.4: %q", got)
	}
	got = ssidFromConnmanPath("/net/connman/service/wifi_000af51c1ea1_4d494e4820544849454e_managed_psk")
	if got != "MINH THIEN" {
		t.Fatalf("MINH THIEN: %q", got)
	}
	got = ssidFromConnmanPath("wifi_000af51c1ea0_746f73686962615f64625f38303333_managed_none")
	if got != "toshiba_db_8033" {
		t.Fatalf("toshiba: %q", got)
	}
	got = ssidFromConnmanPath("wifi_000af51c1ea1_5875616e20416e_managed_psk")
	if got != "Xuan An" {
		t.Fatalf("Xuan An: %q", got)
	}
}

func TestParseConnmanctlKeepsSpaces(t *testing.T) {
	out := "" +
		"*AO Huynh 2.4            wifi_000af51c1ea0_4875796e6820322e34_managed_psk\n" +
		"*A  Huynh 2.4            wifi_000af51c1ea1_4875796e6820322e34_managed_psk\n" +
		"    MINH THIEN           wifi_000af51c1ea1_4d494e4820544849454e_managed_psk\n" +
		"    toshiba_db_8033      wifi_000af51c1ea0_746f73686962615f64625f38303333_managed_none\n" +
		"    Xuan An              wifi_000af51c1ea0_5875616e20416e_managed_psk\n"
	nets := parseConnmanctl(out)
	got := map[string]bool{}
	for _, n := range nets {
		got[n.SSID] = true
	}
	for _, want := range []string{"Huynh 2.4", "MINH THIEN", "toshiba_db_8033", "Xuan An"} {
		if !got[want] {
			t.Fatalf("missing %q in %#v", want, got)
		}
	}
	if got["2.4"] || got["THIEN"] || got["An"] || got["_db_8033"] {
		t.Fatalf("truncated leftovers: %#v", got)
	}
}

func TestMergeWifiNetsPromotesShortSuffix(t *testing.T) {
	out := mergeWifiNets(
		[]wifiNet{{SSID: "2.4", Signal: 80, Secure: true}},
		[]wifiNet{{SSID: "Huynh 2.4", Signal: 40, Secure: true}},
	)
	if len(out) != 1 || out[0].SSID != "Huynh 2.4" {
		t.Fatalf("%+v", out)
	}
	if out[0].Signal != 80 {
		t.Fatalf("keep stronger signal: %+v", out[0])
	}
	out = mergeWifiNets(
		[]wifiNet{{SSID: "_db_8033", Signal: 10, Secure: false}},
		[]wifiNet{{SSID: "toshiba_db_8033", Signal: 20, Secure: false}},
	)
	if len(out) != 1 || out[0].SSID != "toshiba_db_8033" {
		t.Fatalf("underscore truncate: %+v", out)
	}
}
