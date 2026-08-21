#!/bin/bash

ssh -i ~/ssh_root_key root@$1 "systemctl stop wired && mount -o rw,remount /"

scp -i ~/ssh_root_key ./build/wired root@$1:/usr/bin/
scp -i ~/ssh_root_key ./scripts/vic-setup-ap root@$1:/usr/bin/vic-setup-ap
ssh -i ~/ssh_root_key root@$1 "chmod 755 /usr/bin/vic-setup-ap && cp -f /usr/bin/vic-setup-ap /anki/bin/vic-setup-ap && chmod 755 /anki/bin/vic-setup-ap"

scp -i ~/ssh_root_key -r ./webroot/* root@$1:/etc/wired/webroot/

#rsync -e "ssh -i ~/ssh_root_key" -avr ./modfiles/* root@$1:/etc/wired/mods/

ssh -i ~/ssh_root_key root@$1 "
killall -9 vic-on-exit 2>/dev/null
rm -f /run/wireos-wifi-busy /run/wireos-wifi-grace /run/wireos-wifi-hold-ap /run/wireos-wifi-trying /run/wireos-wifi-pending.json /run/wireos-wifi-boot-wait /run/wireos-force-ap
systemctl start wired
"
