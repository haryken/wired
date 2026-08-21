# Vector WiFi hotspot (setup portal)

Trial trên save branch: **chỉ luồng hotspot** (không join từ LAN `:8080`).

Khi mất WiFi nhà, robot phát **hotspot mở** (không mật khẩu) — bấm Kết nối trên phone là vào.

> ConnMan tethering của Anki **bắt buộc WPA**. Hotspot WireOS dùng `wpa_supplicant` AP `key_mgmt=NONE` (`vic-setup-ap`).

## Luồng

1. Không có WiFi client ~45 giây (và hết boot grace nếu đã có WiFi đã lưu) → `vic-switchboard` / `wired` gọi `vic-setup-ap on`.
2. SSID = tên robot (`Vector C3P0`…). **Không mật khẩu.**
3. Phone nối → captive hoặc `http://10.3.141.1/wifi`.
4. Quét/gõ SSID nhà **2.4 GHz** + mật khẩu ≥ 8 → Áp dụng.
5. Hotspot tắt, ConnMan vào WiFi nhà. Sai mật khẩu → hotspot bật lại.
6. Web sau khi online: `http://<ip>:8080/`

## File

| Chỗ | Việc |
|-----|------|
| `anki/wired/scripts/vic-setup-ap` | AP mở + DHCP `10.3.141.1` |
| `wifiWatcher.cpp` | Gọi script khi offline (+ boot grace) |
| `wifi-setup.go` + `wifi-join-hotspot.go` | Portal `/wifi` + join từ hotspot |

## Deploy

`wired`: `make && ./send_to_bot.sh <ip>` (binary, webroot, `vic-setup-ap` → `/usr/bin` và `/anki/bin`).  
`vic-switchboard` + `vic-anim`: scp binary mới (AP auto + icon mặt).
