TARGET ?= esp32s3-box-3
# BLE 用ターゲット。TinyGo 0.43 以降はターゲットに espradio ビルドタグと
# ESP32-S3 の BT ROM シンボルが含まれるので通常のターゲットでよい。
# それより古い TinyGo (0.42.0-dev など) ではどちらも無いため、アップストリームの
# esp32s3.ld を同梱したカスタムターゲット (targets/) を使う。
# `tinygo info` の build tags に espradio があるかで自動判定する。
# リンカスクリプトのパスは firmware/ (make を実行するディレクトリ) からの相対。
# BLE_DEBUG=1 で bluetooth パッケージのデバッグ出力 (ATT/HCI の処理) を有効にする
BLE_DEBUG ?=
BLE_NATIVE := $(shell tinygo info --target $(TARGET) 2>/dev/null | grep -c espradio)
ifeq ($(BLE_NATIVE),0)
TARGET_BLE ?= ./targets/esp32s3-box-3-ble.json
BLE_TAGLIST = espradio
else
TARGET_BLE ?= $(TARGET)
BLE_TAGLIST =
endif
ifneq ($(BLE_DEBUG),)
BLE_TAGLIST += bledebug
endif
BLE_TAGS = $(if $(strip $(BLE_TAGLIST)),-tags "$(strip $(BLE_TAGLIST))",)
EXAMPLES := $(notdir $(wildcard examples/*))
BLE_EXAMPLES := $(filter ble-%,$(EXAMPLES))
OTHER_EXAMPLES := $(filter-out ble-%,$(EXAMPLES))

# espradio (Wi-Fi / BLE) の C コードが要求するフラグを CGo に許可する
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
	@for e in $(OTHER_EXAMPLES); do \
		echo "== $$e"; \
		tinygo build --target $(TARGET) --size short -o /dev/null ./examples/$$e || exit 1; \
	done
	@for e in $(BLE_EXAMPLES); do \
		echo "== $$e"; \
		tinygo build --target $(TARGET_BLE) $(BLE_TAGS) --size short -o /dev/null ./examples/$$e || exit 1; \
	done

# 例: make flash-wifi-server SSID=yourssid PASS=yourpassword
flash-wifi-%:
	@test -n "$(SSID)" || { echo "SSID is empty: set SSID/PASS or WIFI_SSID/WIFI_PASS"; exit 1; }
	tinygo flash --target $(TARGET) --size short -ldflags="$(WIFI_LDFLAGS)" ./examples/wifi-$*

# 例: make flash-ble-sensor
flash-ble-%:
	tinygo flash --target $(TARGET_BLE) $(BLE_TAGS) --size short ./examples/ble-$*

# 例: make flash-blink
flash-%:
	tinygo flash --target $(TARGET) --size short ./examples/$*

# 例: make monitor
monitor:
	tinygo monitor --target $(TARGET)
