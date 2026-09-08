package mods

import "testing"

func TestSSIDFromConnmanPathKeepsFullName(t *testing.T) {
	tests := map[string]string{
		"wifi_000af51c1ea0_4875796e6820322e34_managed_psk":                        "Huynh 2.4",
		"/net/connman/service/wifi_000af51c1ea1_4d494e4820544849454e_managed_psk": "MINH THIEN",
		"wifi_000af51c1ea0_746f73686962615f64625f38303333_managed_none":           "toshiba_db_8033",
		"wifi_000af51c1ea1_5875616e20416e_managed_psk":                            "Xuan An",
	}
	for path, want := range tests {
		if got := ssidFromConnmanPath(path); got != want {
			t.Errorf("ssidFromConnmanPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestParseConnmanctlKeepsFullNames(t *testing.T) {
	out := "" +
		"*AO Huynh 2.4       wifi_000af51c1ea0_4875796e6820322e34_managed_psk\n" +
		"    MINH THIEN      wifi_000af51c1ea1_4d494e4820544849454e_managed_psk\n" +
		"    toshiba_db_8033 wifi_000af51c1ea0_746f73686962615f64625f38303333_managed_none\n" +
		"    Xuan An         wifi_000af51c1ea0_5875616e20416e_managed_psk\n"
	nets := parseConnmanctl(out)
	got := make(map[string]bool, len(nets))
	for _, network := range nets {
		got[network.SSID] = true
	}
	for _, want := range []string{"Huynh 2.4", "MINH THIEN", "toshiba_db_8033", "Xuan An"} {
		if !got[want] {
			t.Errorf("missing full SSID %q in %#v", want, got)
		}
	}
	for _, bad := range []string{"2.4", "THIEN", "_db_8033", "An"} {
		if got[bad] {
			t.Errorf("found truncated SSID %q in %#v", bad, got)
		}
	}
}

func TestMergeWifiNetsDropsTruncatedSuffixes(t *testing.T) {
	nets := mergeWifiNets(
		[]wifiNet{{SSID: "2.4", Signal: 80, Secure: true}, {SSID: "_db_8033", Signal: 10}},
		[]wifiNet{{SSID: "Huynh 2.4", Signal: 40, Secure: true}, {SSID: "toshiba_db_8033", Signal: 20}},
	)
	got := make(map[string]wifiNet, len(nets))
	for _, network := range nets {
		got[network.SSID] = network
	}
	if got["Huynh 2.4"].Signal != 80 {
		t.Errorf("stronger signal was not retained: %#v", got)
	}
	if _, ok := got["toshiba_db_8033"]; !ok {
		t.Errorf("full underscore SSID missing: %#v", got)
	}
	if _, ok := got["2.4"]; ok {
		t.Errorf("truncated space suffix retained: %#v", got)
	}
	if _, ok := got["_db_8033"]; ok {
		t.Errorf("truncated underscore suffix retained: %#v", got)
	}
}
