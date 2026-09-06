# Vector WiFi hotspot (setup portal)

Trial trên save branch: **chỉ luồng hotspot** (không join từ LAN `:8080`).

Khi mất WiFi nhà, robot mở **đúng chế độ đã chọn** (mặc định hotspot). Chế độ lưu tại `/data/wired/wifi-setup-mode`. Mặt: **WIFI BY BLUE** / **WIFI HOTSPOT** để đổi.

> ConnMan tethering của Anki **bắt buộc WPA**. Hotspot WireOS dùng `wpa_supplicant` AP `key_mgmt=NONE` (`vic-setup-ap`).

## Luồng

1. Không có WiFi nhà (hết boot grace nếu đã lưu mạng) → hotspot **hoặc** BLE pairing, theo mode đã nhớ.
2. Hotspot: SSID = tên robot (`Vector C3P0`…). **Không mật khẩu.** Phone nối → OS captive portal **tự mở** trang setup (DNS `* → 192.168.4.1` + HTTP 302). Dự phòng: `http://192.168.4.1/` hoặc `:8080`.
3. BLE: pairing Bluetooth như Vector cũ (ấn lưng 2 lần cũng vào BLE nếu đang chọn mode đó, hoặc đang có WiFi nhà).
4. Quét/gõ SSID nhà **2.4 GHz** + mật khẩu ≥ 8 → Áp dụng.
5. Hotspot tắt, ConnMan vào WiFi nhà. Sai mật khẩu → hotspot bật lại (nếu mode hotspot).
6. Web sau khi online: `http://<ip>:8080/`

**Boot race (fixed):** Khi vừa associate `Huynh 2.4` nhưng DHCP chưa xong (IP `169.254.x`), **không** được bật setup AP (sẽ kill ConnMan). Wired chờ DHCP ~75s; WifiWatcher coi ConnMan `association`/`configuration` như đang nối.

## Captive auto-open (giống Xiaozhi)

| Thành phần | Việc |
|------------|------|
| `dnsmasq` (nếu có trong image) | DHCP + `--address=/#/192.168.4.1` |
| `wired` `captive-dns.go` | Fallback UDP :53 → A=`192.168.4.1` khi không có dnsmasq |
| `WrapCaptive` + probe routes | OS check (`generate_204`, `hotspot-detect.html`, …) → 302 portal |

## File

| Chỗ | Việc |
|-----|------|
| `anki/wired/scripts/vic-setup-ap` | AP mở + DHCP `192.168.4.1` |
| `wifiWatcher.cpp` | Gọi script khi offline (+ boot grace) |
| `wifi-setup.go` + `wifi-join-hotspot.go` + `captive-dns.go` | Portal `/wifi` + join + DNS hijack |

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
