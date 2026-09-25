TARGET ?= esp32s3-box-3
EXAMPLES := $(notdir $(wildcard examples/*))

# espradio (Wi-Fi) の C コードが要求するフラグを CGo に許可する
export CGO_CFLAGS_ALLOW = -fno-short-enums

# Wi-Fi の example 用。次のどちらでも指定できる。
#   make flash-wifi-server SSID=yourssid PASS=yourpassword
#   export WIFI_SSID=yourssid WIFI_PASS=yourpassword; make flash-wifi-server
# (環境変数 SSID / PASS でも可)
SSID ?= $(WIFI_SSID)
PASS ?= $(WIFI_PASS)
WIFI_LDFLAGS = -X main.ssid=$(SSID) -X main.password=$(PASS)

.PHONY: all build monitor flash-%

all: build

# 全 example をビルドして生成物は捨てる (コンパイル確認用)
build:
	@for e in $(EXAMPLES); do \
		echo "== $$e"; \
		tinygo build --target $(TARGET) --size short -o /dev/null ./examples/$$e || exit 1; \
	done

# 例: make flash-wifi-server SSID=yourssid PASS=yourpassword
flash-wifi-%:
	@test -n "$(SSID)" || { echo "SSID is empty: set SSID/PASS or WIFI_SSID/WIFI_PASS"; exit 1; }
	tinygo flash --target $(TARGET) --size short -ldflags="$(WIFI_LDFLAGS)" ./examples/wifi-$*

# 例: make flash-blink
flash-%:
	tinygo flash --target $(TARGET) --size short ./examples/$*

# 例: make monitor
monitor:
	tinygo monitor --target $(TARGET)
