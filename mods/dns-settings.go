package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/os-vector/wired/vars"
)

// DNS mode for outbound resolve (AI HTTPS/WSS, updates, …).
// Default "dhcp" = router DHCP nameservers (current Vector behavior).
// "public" = prefer 8.8.8.8 then 1.1.1.1 via ConnMan Nameservers.Configuration.
const (
	dnsModeDHCP   = "dhcp"
	dnsModePublic = "public"

	dnsPublicPrimary   = "8.8.8.8"
	dnsPublicSecondary = "1.1.1.1"
)

var dnsConfigPath = "/data/data/com.anki.victor/persistent/wired/dns.json"

type dnsCfg struct {
	Mode string `json:"mode"`
}

type DnsSettings struct {
	vars.Modification
}

func NewDnsSettings() *DnsSettings {
	return &DnsSettings{}
}

func (m *DnsSettings) Name() string { return "DnsSettings" }

func (m *DnsSettings) Description() string {
	return "Switch between router DHCP DNS and public DNS (8.8.8.8 / 1.1.1.1) for AI connectivity"
}

func (m *DnsSettings) Load() error {
	go dnsReconcileLoop()
	return nil
}

func (m *DnsSettings) HTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case vars.IsEndpoint(r, "get"):
		cfg := loadDnsCfg()
		view := dnsStatusView(cfg)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
		return
	case vars.IsEndpoint(r, "set"):
		mode := strings.ToLower(strings.TrimSpace(r.FormValue("mode")))
		if mode != dnsModeDHCP && mode != dnsModePublic {
			vars.HTTPError(w, r, "mode must be dhcp or public")
			return
		}
		cfg := dnsCfg{Mode: mode}
		if err := saveDnsCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		view := dnsStatusView(cfg)
		// During setup AP there is no client Wi-Fi — save preference only;
		// dnsApplySaved() / reconcile apply after join.
		if dnsShouldSkipApply() {
			view["applied"] = true
			view["deferred"] = true
		} else if applyErr := dnsApplyMode(mode); applyErr != nil {
			view["apply_error"] = applyErr.Error()
			view["applied"] = false
		} else {
			view["applied"] = true
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": view,
		})
		return
	}
	vars.HTTPError(w, r, "404 not found")
}

func dnsStatusView(cfg dnsCfg) map[string]interface{} {
	mode := normalizeDnsMode(cfg.Mode)
	svc, resolv, names := dnsInspect()
	return map[string]interface{}{
		"mode":             mode,
		"service":          svc,
		"resolv_conf":      resolv,
		"active_nameservers": names,
		"public_primary":   dnsPublicPrimary,
		"public_secondary": dnsPublicSecondary,
	}
}

func normalizeDnsMode(mode string) string {
	if strings.ToLower(strings.TrimSpace(mode)) == dnsModePublic {
		return dnsModePublic
	}
	return dnsModeDHCP
}

func loadDnsCfg() dnsCfg {
	cfg := dnsCfg{Mode: dnsModeDHCP}
	data, err := os.ReadFile(dnsConfigPath)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	cfg.Mode = normalizeDnsMode(cfg.Mode)
	return cfg
}

func saveDnsCfg(cfg dnsCfg) error {
	cfg.Mode = normalizeDnsMode(cfg.Mode)
	if err := os.MkdirAll(filepath.Dir(dnsConfigPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dnsConfigPath, data, 0644)
}

// dnsApplySaved is called after Wi-Fi join so public mode survives reconnect/reboot.
func dnsApplySaved() {
	mode := loadDnsCfg().Mode
	if mode != dnsModePublic {
		return
	}
	if err := dnsApplyMode(mode); err != nil {
		wifiLog("dns apply saved mode=%s: %v", mode, err)
	}
}

func dnsApplyMode(mode string) error {
	mode = normalizeDnsMode(mode)
	if dnsShouldSkipApply() {
		return fmt.Errorf("skip while setup AP / no client Wi-Fi")
	}
	svc := dnsActiveWifiService()
	if svc == "" {
		return fmt.Errorf("no active wifi service")
	}
	if mode == dnsModePublic {
		if err := dnsSetNameservers(svc, []string{dnsPublicPrimary, dnsPublicSecondary}); err != nil {
			return err
		}
		wifiLog("dns public applied svc=%s %s %s", svc, dnsPublicPrimary, dnsPublicSecondary)
		return nil
	}
	if err := dnsSetNameservers(svc, nil); err != nil {
		return err
	}
	wifiLog("dns dhcp restored svc=%s", svc)
	return nil
}

func dnsShouldSkipApply() bool {
	// Captive setup AP owns :53 — never override client DNS while tethered.
	if hsAPOn() {
		return true
	}
	return false
}

func dnsActiveWifiService() string {
	bus, err := dbus.SystemBus()
	if err != nil {
		return dnsActiveWifiServiceCtl()
	}
	manager := bus.Object("net.connman", "/")
	var services [][]interface{}
	if err := manager.Call("net.connman.Manager.GetServices", 0).Store(&services); err != nil {
		return dnsActiveWifiServiceCtl()
	}
	var ready string
	for _, pair := range services {
		if len(pair) < 2 {
			continue
		}
		path, _ := pair[0].(dbus.ObjectPath)
		props, ok := pair[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		if typ, _ := props["Type"].Value().(string); typ != "wifi" {
			continue
		}
		if hsEthernetIface(props) != "" && hsEthernetIface(props) != "wlan0" {
			continue
		}
		st, _ := props["State"].Value().(string)
		id := strings.TrimPrefix(string(path), "/net/connman/service/")
		if id == "" {
			continue
		}
		if st == "online" {
			return id
		}
		if st == "ready" && ready == "" {
			ready = id
		}
	}
	if ready != "" {
		return ready
	}
	return dnsActiveWifiServiceCtl()
}

func dnsActiveWifiServiceCtl() string {
	out, err := exec.Command("connmanctl", "services").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "*AO") && !strings.Contains(line, "*AR") && !strings.Contains(line, "*A ") {
			continue
		}
		_, svc := hsParseCtl(line)
		if strings.HasPrefix(svc, "wifi_") {
			return svc
		}
	}
	return ""
}

func dnsSetNameservers(svc string, ns []string) error {
	if svc == "" {
		return fmt.Errorf("empty service")
	}
	// Prefer D-Bus Nameservers.Configuration (empty array = back to DHCP DNS).
	if err := dnsSetNameserversDBus(svc, ns); err == nil {
		return nil
	} else if len(ns) == 0 {
		// Fallback clear via connmanctl with no args.
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		out, err2 := exec.CommandContext(ctx, "connmanctl", "config", svc, "--nameservers").CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("dbus: %v; connmanctl: %v (%s)", err, err2, strings.TrimSpace(string(out)))
		}
		return nil
	}
	args := []string{"config", svc, "--nameservers"}
	args = append(args, ns...)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "connmanctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("connmanctl: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dnsSetNameserversDBus(svc string, ns []string) error {
	bus, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	obj := bus.Object("net.connman", dbus.ObjectPath("/net/connman/service/"+svc))
	if ns == nil {
		ns = []string{}
	}
	call := obj.Call("net.connman.Service.SetProperty", 0, "Nameservers.Configuration", dbus.MakeVariant(ns))
	return call.Err
}

func dnsInspect() (svc, resolv string, names []string) {
	svc = dnsActiveWifiService()
	b, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		b, _ = os.ReadFile("/data/run/connman/resolv.conf")
	}
	resolv = strings.TrimSpace(string(b))
	for _, line := range strings.Split(resolv, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver ") {
			names = append(names, strings.TrimSpace(strings.TrimPrefix(line, "nameserver ")))
		}
	}
	return svc, resolv, names
}

var (
	dnsReconcileOnce sync.Once
)

func dnsReconcileLoop() {
	dnsReconcileOnce.Do(func() {
		// Boot: ConnMan may still be associating — retry a few times.
		for i := 0; i < 12; i++ {
			time.Sleep(5 * time.Second)
			if loadDnsCfg().Mode != dnsModePublic {
				break
			}
			if dnsShouldSkipApply() {
				continue
			}
			if err := dnsApplyMode(dnsModePublic); err == nil {
				break
			}
		}
		t := time.NewTicker(45 * time.Second)
		defer t.Stop()
		for range t.C {
			if loadDnsCfg().Mode != dnsModePublic {
				continue
			}
			if dnsShouldSkipApply() {
				continue
			}
			// Re-apply if ConnMan DHCP rewrote resolv without public DNS.
			_, _, names := dnsInspect()
			if dnsHasPublic(names) {
				continue
			}
			_ = dnsApplyMode(dnsModePublic)
		}
	})
}

func dnsHasPublic(names []string) bool {
	hasPrimary := false
	for _, n := range names {
		if n == dnsPublicPrimary {
			hasPrimary = true
		}
	}
	return hasPrimary
}
