package main

import "testing"

func TestPortalAPIAllowlist(t *testing.T) {
	allowed := []string{
		"/api/mods/JdocSettings/getVolume",
		"/api/mods/Xiaozhi/get",
		"/api/mods/Control/cam-stream",
		"/api/extra/restartvic",
	}
	for _, path := range allowed {
		if !portalAPIAllowed(path) {
			t.Errorf("expected %q to be allowed", path)
		}
	}

	blocked := []string{
		"/api/mods/AutoUpdate/start",
		"/api/mods/WifiSetup/connect",
		"/api/extra/restartvic-unsafe",
		"/not-api",
	}
	for _, path := range blocked {
		if portalAPIAllowed(path) {
			t.Errorf("expected %q to be blocked", path)
		}
	}
}
