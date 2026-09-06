# Vector WiFi hotspot (setup portal)

Trial trên save branch: **chỉ luồng hotspot** (không join từ LAN `:8080`).

Khi mất WiFi nhà, robot mở **đúng chế độ đã chọn** (mặc định hotspot). Chế độ lưu tại `/data/wired/wifi-setup-mode`. Mặt: **WIFI BY BLUE** / **WIFI HOTSPOT** để đổi.

> ConnMan tethering của Anki **bắt buộc WPA**. Hotspot WireOS dùng `wpa_supplicant` AP `key_mgmt=NONE` (`vic-setup-ap`).

## Luồng

1. Không có WiFi nhà (hết boot grace nếu đã lưu mạng) → hotspot **hoặc** BLE pairing, theo mode đã nhớ.
2. Hotspot: SSID = tên robot (`Vector C3P0`…). **Không mật khẩu.** Portal `http://192.168.4.1/` hoặc `http://192.168.4.1:8080/` (nếu :80 bị chặn bản cũ).
3. BLE: pairing Bluetooth như Vector cũ (ấn lưng 2 lần cũng vào BLE nếu đang chọn mode đó, hoặc đang có WiFi nhà).
4. Quét/gõ SSID nhà **2.4 GHz** + mật khẩu ≥ 8 → Áp dụng.
5. Hotspot tắt, ConnMan vào WiFi nhà. Sai mật khẩu → hotspot bật lại (nếu mode hotspot).
6. Web sau khi online: `http://<ip>:8080/`

## File

| Chỗ | Việc |
|-----|------|
| `anki/wired/scripts/vic-setup-ap` | AP mở + DHCP `192.168.4.1` |
| `wifiWatcher.cpp` | Gọi script khi offline (+ boot grace) |
| `wifi-setup.go` + `wifi-join-hotspot.go` | Portal `/wifi` + join từ hotspot |

## Deploy

`wired`: OTA installs `wired` + `vic-setup-ap`. Hot-deploy: `make && ./send_to_bot.sh <ip>` (binary, webroot, `vic-setup-ap` → `/usr/bin` và `/anki/bin`).  
`vic-switchboard` + `vic-anim`: scp binary mới (AP auto + icon mặt).

## Known failure (fixed)

Stock Vector iptables: `INPUT DROP`, DHCP only on iface `tether`, web only `:8080`.
WireOS open AP is on `wlan0` → phone associated but **got no IP**, so neither
`http://192.168.4.1` nor `:8080` worked. Also busybox `udhcpd` needs a writable
`lease_file` or it exits immediately.

Fix: open DHCP/DNS/80 in iptables; `vic-setup-ap` opens firewall **before** DHCP,
writes lease file, verifies udhcpd stays up.
