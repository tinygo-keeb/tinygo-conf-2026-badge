# Firmware

[日本語](README.md)

TinyGo firmware for the TinyGo Conference 2026 badge, based on the ESP32-S3-DevKit.

## Build and flash

From the repository root:

```sh
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/blink
tinygo monitor --target esp32s3-box-3
```

The `smoketest` target in `Makefile` lists the build command for each example on its own line. Run `make smoketest` to build them all into `out/`. To flash an example, copy its line and replace `build -o ./out/xxx.bin` with `flash`.

Wi-Fi examples embed the SSID and password at build time. The Makefile uses `YOUR_SSID` and `YOUR_PASSWORD` placeholders. Replace them with your values, or use `make flash-wifi-<name>` to supply credentials through Make variables or environment variables:

```sh
make flash-wifi-server SSID=yourssid PASS=yourpassword
# Environment variables also work (WIFI_SSID / WIFI_PASS, or SSID / PASS).
export WIFI_SSID=yourssid WIFI_PASS=yourpassword
make flash-wifi-server
# Manual command:
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short \
  -ldflags="-X main.ssid=yourssid -X main.password=yourpassword" ./examples/wifi-server
```

With TinyGo 0.43 or later, BLE examples build with the standard target used in the Makefile. Older TinyGo versions need the custom target `targets/esp32s3-box-3-ble.json` and `-tags espradio`. Run these commands from `firmware/` because the linker script path is relative to that directory. Add `-tags bledebug` to print Bluetooth ATT/HCI debug output over serial.

```sh
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short ./examples/ble-sensor
# For older TinyGo versions:
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target ./targets/esp32s3-box-3-ble.json \
  -tags espradio --size short ./examples/ble-sensor
```

## Layout

| Path | Description |
| --- | --- |
| `badge/` | Pin assignments and helpers for initializing peripherals |
| `ws2812s3/` | WS2812B driver for the ESP32-S3 at 240 MHz |
| `i2s/` | 16-bit stereo audio output driver using ESP32-S3 I2S0 and GDMA |
| `wifi/` | Helper for connecting through espradio; resets and retries after a failure |
| `flashstore/` | Settings storage that reads and writes 64 KB of SPI flash starting at 0x1F0000 through ROM functions |
| `targets/` | Custom BLE target with an upstream TinyGo esp32s3.ld linker script |
| `examples/blink` | Cycle the WS2812B LEDs through rainbow colors |
| `examples/all` | Launcher for all 20 examples: logo on startup, SW1 opens the menu, joystick up/down selects, SW1 or joystick press starts, left returns from an example to the menu, and left in the menu returns to TOP. See [the launcher README](examples/all/README.md) for build commands and input changes. |
| `examples/display` | Show color bars and text on the ST7789 |
| `examples/input` | Print the SW1/SW2 and joystick states over serial |
| `examples/joyraw` | Measure raw joystick values and movement limits (min/max) |
| `examples/aht21b` | Print temperature and humidity readings |
| `examples/i2cscan` | Scan the Grove and AHT21B I2C buses |
| `examples/dht20` | Print readings from a Grove-connected DHT20 (AHT20-compatible) sensor |
| `examples/wifi-httpget` | Connect to Wi-Fi and fetch http://httpbin.org/get using net/http |
| `examples/wifi-server` | Serve AHT21B temperature and humidity readings over HTTP using httphi |
| `examples/wifi-joystick` | Show joystick X/Y and switch states live in a browser |
| `examples/ble-scanner` | Scan nearby BLE devices and print addresses, RSSI, and names |
| `examples/ble-sensor` | BLE peripheral with temperature and humidity (Environmental Sensing), color writes for two WS2812B LEDs, button notifications, and automatic/manual LED modes. Shows the chip-specific ID in its name and on the LCD |
| `examples/ble-sensor/webble.html` | Web Bluetooth control page for the BLE peripheral; open it in Chrome or Edge. It is also published at https://conf.tinygo-keeb.org/2026/conf2026badge/ |
| `examples/ir` | Receive NEC infrared signals and send one when a button is pressed |
| `examples/irlearn` | Learn, list, name, delete, save, and retransmit remote-control signals |
| `examples/audio` | Play notes, melodies, and beeps through the MAX98357 |
| `examples/audiotest` | Print I2S diagnostics and continuously play a 1 kHz sine wave |
| `examples/demo` | Full-feature demo combining the features above |
| `examples/rhythm` | Rhythm game: press five controls (joystick left, up or down, right, SW2, and SW1) as notes fall. The BGM is synthesized live; pitch corresponds to lane, and effects grow with the combo. EASY / NORMAL modes |
| `examples/slotgame` | Three-reel slot game controlled entirely with button_1 (SW1). Press to start, then press again to stop each reel from left to right. See [the game README](examples/slotgame/README.md) for controls and build commands. |
| `examples/selftest` | Check all onboard devices together. Buttons, joystick directions, AHT21B, infrared loopback, and I2S are checked automatically; the display shows ALL OK when complete. Check the LCD, LEDs, speaker, and Grove by sight or sound |

## Pin assignments

These assignments come from `../hardware/tinygo-conf-2026.kicad_sch`. Constants are in `badge/badge.go`.

| Function | Signal | GPIO | Notes |
| --- | --- | --- | --- |
| LCD ST7789 (J3) | SCL | 12 | SPI0 (FSPI) |
| | SDA (MOSI) | 11 | |
| | CS | 10 | |
| | DC | 5 | |
| | RES | 4 | |
| | BLK | - | Connected directly to 3V3 |
| I2S MAX98357 (U3) | BCLK | 45 | I2S0 (see below) |
| | LRC | 21 | |
| | DIN | 47 | |
| WS2812B x2 (D1, D2) | DIN | 16 | D1 -> D2 in series |
| Joystick (U2) | X | 7 | ADC1_CH6 |
| | Y | 6 | ADC1_CH5 |
| | BTN | 15 | Grounded when pressed; internal pull-up |
| SW1 | | 13 | Grounded when pressed; internal pull-up |
| SW2 | | 14 | Grounded when pressed; internal pull-up |
| IR LED (D5) | | 17 | High turns it on; modulated with 38 kHz PWM |
| IR receiver (J2) | DATA | 18 | |
| Grove I2C (J1) | SDA | 8 | I2C0 |
| | SCL | 9 | |
| AHT21B (J4) | SDA | 41 | I2C1 |
| | SCL | 42 | |

Expansion header J5 (2x8):

```
 1: 3V3     2: 3V3
 3: GPIO1   4: GPIO2
 5: GPIO3   6: GPIO38
 7: GPIO39  8: GND
 9: GND    10: GPIO40
11: GPIO41 12: GPIO42   (shared with AHT21B SDA/SCL)
13: GPIO45 14: GPIO46   (GPIO45 is shared with I2S BCLK)
15: GPIO48 16: VCC (5V)
```

## Notes

- **Target:** Use `esp32s3-box-3`. The `esp32s3-generic` target does not define
  `machine.CPUFrequency()` or the default SPI pins, so some drivers cannot build.
- **WS2812B:** On Xtensa, `tinygo.org/x/drivers/ws2812` only supports 80 and 160 MHz.
  The ESP32-S3 runs at 240 MHz, so `ws2812s3/` provides a bit-banging driver
  with timing for 240 MHz.
- **LCD flicker:** Drawing directly to the display makes individual updates
  visible. `badge.NewFramebuffer(display)` creates a full-screen offscreen
  buffer (RGB565, about 115 KB). Draw there, then call `Display()` to transfer
  it in one pass (see `examples/demo`). The buffer can also be passed directly
  to tinyfont and tinydraw.
- **I2C initialization:** During its final bus-clear step,
  `machine.I2C.Configure` waits indefinitely for a completion bit and may hang
  on a floating bus without pull-ups. Configure also overwrites pin settings,
  removing any internal pull-ups enabled beforehand. `badge.ConfigureI2C()`
  implements the same sequence with internal pull-up support and a finite
  wait. Both Grove (`ConfigureGroveI2C`, with internal pull-ups at 100 kHz)
  and AHT21B (`ConfigureSensorI2C`) use it.
- **I2C scanning:** TinyGo's ESP32 I2C driver does not report an address NACK
  as an error on reads. On writes, after a NACK it leaves unsent data in the
  TX FIFO, causing the next transaction to become a general call, which the
  AHT21B acknowledges. `badge.ProbeI2C()` therefore sends only the address
  byte through direct register access, then calls `badge.ResetI2C()` to reset
  the FSM and FIFO after a failure.
- **Joystick travel:** The joystick cap limits its movement. Raw ADC readings
  travel only about ±16000 to 17000 counts around the center, roughly half the
  full 0..65535 range. `badge.Joystick` normalizes with `Range` (default 16000),
  so a full deflection reaches 1000. If the range differs, measure it with
  `examples/joyraw` and adjust `Range`. Stationary noise is about ±100 counts,
  so `DeadZone` is set to 50 (equivalent to 800 counts).
- **USB serial:** Output from `print` is not sent until a newline. Long
  operations without a newline may appear to produce no output; use `println`
  at checkpoints.
- **Output in interrupts:** Callbacks such as those in infrared reception
  (irremote) run inside GPIO interrupts. Calling `println` there can stall USB
  serial output. Save the data in the callback and print it from the main
  loop instead (see `examples/ir`).
- **Flash storage:** TinyGo's machine package has no ESP32-S3 flash API.
  `flashstore/` calls the fixed-address ROM functions
  `esp_rom_spiflash_read/write/erase_sector/unlock` directly through CGo;
  the addresses come from ESP-IDF v5.1.2's esp32s3.rom.ld. It follows the
  same procedure as TinyGo's ESP32-C3 driver and calls the ROM functions
  with interrupts disabled. TinyGo writes 2 MB as the flash size in the image
  header, so the ROM driver rejects erase/write operations above 2 MB.
  The settings area therefore occupies the final 64 KB below 2 MB,
  starting at 0x1F0000. Reads also use ROM functions instead of the cache.
- **I2S (MAX98357):** TinyGo's machine package does not support ESP32-S3 I2S.
  The implementation in `i2s/` accesses registers directly, following the
  procedure from ESP-IDF v5.1's i2s_ll.h, gdma_ll.h, and i2s_std.c. ESP32-S3
  I2S can send data only through GDMA, so DMA buffers form a ring for
  continuous playback; `Write` fills them in turn. The ring keeps playing,
  so call `Silence()` (or `Stop()` on a `badge.ToneGenerator`) after playback.
  The format is Philips standard, 16-bit, two channels, MCLK = fs*256, and
  BCLK = fs*32 (16-bit slots; `Config.SlotBits` also allows 32). GDMA uses
  channel 0.
- **Wi-Fi:** The firmware uses `tinygo.org/x/espradio` (TinyGo 0.41 or later),
  which combines Espressif binary blobs with lneto, a pure-Go TCP/IP stack.
  The espradio C code requires `-fno-short-enums`, so builds need
  `CGO_CFLAGS_ALLOW=-fno-short-enums`; relevant Makefile lines set it.
  Embed the SSID and password with
  `-ldflags="-X main.ssid=... -X main.password=..."`. Without them, the
  example repeatedly reports `failure: ssid is empty` at startup. Wi-Fi
  code is then eliminated as dead code, making the binary unusually small.
  The HTTP server uses espradio's recommended `httphi`, which does not
  allocate heap space per request. `net/http` allocates about 10 KB per
  connection and can stall after long runs due to GC fragmentation (see
  espradio's README). Wireless initialization can happen only once: calling
  `NetConnect` again after a failure returns `already enabled`.
  `wifi.Connect()` waits five seconds after a failure, then soft-resets the
  chip with `badge.Reset()` and starts over. Just after closing a monitor,
  a previous session can remain on the access point and cause
  `auth expired`; this retry handles it. Merely importing espradio links
  initialization data even if unused, increasing flash use by about 170 KB
  and RAM use by about 160 KB. For that reason, Wi-Fi helpers live in
  the separate `wifi` package instead of `badge`.
- **BLE:** The firmware uses the espradio backend of
  `tinygo.org/x/bluetooth` (build tag `espradio`). TinyGo 0.43 and later
  apply that tag to ESP32 targets by default and include ESP32-S3 BT ROM
  symbols in the linker script, so the standard target works. Older
  versions, such as 0.42.0-dev, lack both and fail to link because the BLE
  blob (libbtdm_app.a) references about 1,000 ROM symbols, including
  `r_osi_funcs_p`. To support them, this repository includes upstream
  TinyGo's esp32s3.ld (retrieved from the dev branch on 2026-09-25) as
  `targets/esp32s3-ble.ld`. The custom target
  `targets/esp32s3-box-3-ble.json` inherits from esp32s3-box-3 and uses that
  script. With older TinyGo, specify this target and `-tags espradio`.
  The `linkerscript` path in a TinyGo target JSON is resolved relative to
  the current working directory (`firmware/`).
- **Badge identification:** `badge.SerialNumber()` returns a six-digit
  hexadecimal ID from the low three bytes of the chip-specific eFuse MAC
  address. The ble-sensor example advertises as
  `TinyGo Conf 2026 #XXXXXX` and displays the same ID on the LCD, making it
  possible to distinguish badges used together in a workshop.
- **Where espradio is imported:** TinyGo runs package initializers in
  lexicographic import-path order. After an initializer that cannot be
  evaluated at compile time (an espradio C call), subsequent package
  initializers run at runtime instead. This module's import path
  (`github.com/sago35/...`) sorts before `github.com/soypat/lneto` and
  `net/http`. Importing espradio from this module would move Unicode table
  initialization and similar work to runtime, increasing RAM use by about
  70 KB (measured: 115 KB to 240 KB). Therefore `wifi.Connect()` accepts
  a `netlink.Netlinker` rather than importing espradio, and each example's
  main package creates the `Esplink`.
- **MAX98357 SD pin:** The pin is unconnected in the schematic. On an
  Adafruit module, its onboard 1 MΩ pull-up puts SD at about 0.45 V with
  Vin = 5 V (stereo average mode), but some modules did not work in this
  configuration. A module with SD tied directly to Vin (left channel only)
  was tested successfully. The firmware sends the same audio on L and R,
  so both modes sound the same. A future board revision should provide
  the SD pull-up on the board.
