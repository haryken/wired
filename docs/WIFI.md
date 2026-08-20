# Vector WiFi hotspot (setup portal)

Khi mất WiFi nhà, robot phát **hotspot mở** (không mật khẩu) — bấm Kết nối trên phone là vào.

> ConnMan tethering của Anki **bắt buộc WPA**. Hotspot WireOS dùng `wpa_supplicant` AP `key_mgmt=NONE` (`vic-setup-ap`).

## Luồng

1. Không có WiFi client ~45 giây → `vic-switchboard` gọi `vic-setup-ap on`.
2. SSID = tên robot (`Vector C3P0`…). **Không mật khẩu.**
3. Phone nối → captive hoặc `http://10.3.141.1/wifi`.
4. Gõ SSID nhà **2.4 GHz** + mật khẩu ≥ 8 → Áp dụng.
5. Hotspot tắt, ConnMan vào WiFi nhà. Web: `http://<ip>:8080/`

Đã online: tab **WiFi** trên `:80` / `:8080` để đổi mạng.

## File

| Chỗ | Việc |
|-----|------|
| `anki/wired/scripts/vic-setup-ap` | AP mở + DHCP `10.3.141.1` |
| `wifiWatcher.cpp` | Gọi script khi offline |
| `wifi-setup.go` | Portal `/wifi` + tab web |

## Deploy

`wired`: `make && ./send_to_bot.sh <ip>` (copy binary, webroot, **và** `vic-setup-ap`).  
`vic-switchboard`: scp binary mới để tự bật AP.
