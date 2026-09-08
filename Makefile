.PHONY: wired wireos-hotspot-portal

all: wired wireos-hotspot-portal

go_deps:
	echo `${HOME}/.anki/go/dist/1.24.4/go/bin/go version` && ${HOME}/.anki/go/dist/1.24.4/go/bin/go mod download

libvector-gobot:
	cd vector-gobot && make GCC=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang GPP=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang++
	cp vector-gobot/build/libvector-gobot.so build/

wired: go_deps
	echo $(PWD)/vector-gobot/build
	CGO_LDFLAGS="-L$(PWD)/vector-gobot/build/ -Wl,-rpath-link,${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/sysroot/lib -Wl,-rpath-link,${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/sysroot/usr/lib" \
	CGO_CFLAGS="-I$(PWD)/vector-gobot/include" \
	CGO_ENABLED=1 GOARM=7 GOARCH=arm CC=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang \
	CXX=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang++ \
	 ${HOME}/.anki/go/dist/1.24.4/go/bin/go build -tags vicos -ldflags '-w -s' -o build/wired .
	#${HOME}/.anki/upx/dist/5.0.1/upx --best --lzma build/wired

wireos-hotspot-portal: go_deps
	CGO_LDFLAGS="-L$(PWD)/vector-gobot/build/ -Wl,-rpath-link,${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/sysroot/lib -Wl,-rpath-link,${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/sysroot/usr/lib" \
	CGO_CFLAGS="-I$(PWD)/vector-gobot/include" \
	CGO_ENABLED=1 GOARM=7 GOARCH=arm CC=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang \
	CXX=${HOME}/.anki/vicos-sdk/dist/5.3.0-r07/prebuilt/bin/arm-oe-linux-gnueabi-clang++ \
	 ${HOME}/.anki/go/dist/1.24.4/go/bin/go build -tags vicos -ldflags '-w -s' -o build/wireos-hotspot-portal ./cmd/wireos-hotspot-portal


#vic-gateway: go_deps
#	CGO_ENABLED=1 GOARM=7 GOARCH=arm CC=/home/build/.anki/vicos-sdk/dist/4.0.0-r05/prebuilt/bin/arm-oe-linux-gnueabi-clang CXX=/home/build/.anki/vicos-sdk/dist/4.0.0-r05/prebuilt/bin/arm-oe-linux-gnueabi-clang++ PKG_CONFIG_PATH="$(pwd)/armlibs/lib/pkgconfig" CGO_CFLAGS="-I$(pwd)/armlibs/include -I$(pwd)/armlibs/include/opus -I$(pwd)/armlibs/include/ogg" CGO_CXXFLAGS="-stdlib=libc++ -std=c++11" CGO_LDFLAGS="-L$(pwd)/armlibs/lib -L$(pwd)/armlibs/lib/arm-linux-gnueabi/android" /usr/local/go/bin/go build -tags nolibopusfile,vicos -ldflags '-w -s -linkmode internal -extldflags "-static" -r /anki/lib' -o build/vic-gateway gateway/*.go
#
#	upx build/vic-gateway

