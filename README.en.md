# TinyGo Conf 2026 Badge

[日本語](README.md)

This repository contains the hardware design and TinyGo firmware for the TinyGo Conference 2026 badge. Built around an ESP32-S3-DevKit, the badge includes a display, input devices, audio, infrared, and a temperature and humidity sensor.

## Main features

| Feature | Hardware | Firmware examples |
| --- | --- | --- |
| Display | ST7789 display | [`display`](firmware/examples/display), [`demo`](firmware/examples/demo) |
| Input | Joystick and two switches | [`input`](firmware/examples/input), [`rhythm`](firmware/examples/rhythm) |
| Light and sound | Two RGB LEDs, MAX98357 amplifier, and speaker | [`blink`](firmware/examples/blink), [`audio`](firmware/examples/audio) |
| Sensors and expansion | AHT21B temperature and humidity sensor, Grove-compatible I2C connector | [`aht21b`](firmware/examples/aht21b), [`i2cscan`](firmware/examples/i2cscan) |
| Infrared | IR LED and receiver module | [`ir`](firmware/examples/ir), [`irlearn`](firmware/examples/irlearn) |
| Wireless | ESP32-S3 Wi-Fi and BLE | [`wifi-server`](firmware/examples/wifi-server), [`ble-sensor`](firmware/examples/ble-sensor) |

Use [`selftest`](firmware/examples/selftest) to check all onboard devices.

## Assembly

See the [build guide](hardware/build/build.md) for the parts list and illustrated assembly steps. The build guide is in Japanese.

## Documentation and files

| Path | Description |
| --- | --- |
| [hardware/README.en.md](hardware/README.en.md) | Board design, KiCad setup, and manufacturing files |
| [hardware/build/build.md](hardware/build/build.md) | Parts list and illustrated assembly guide (Japanese) |
| [hardware/tinygo-conf-2026.kicad_pro](hardware/tinygo-conf-2026.kicad_pro) | KiCad project; the schematic and PCB layout are in the same directory |
| [firmware/README.en.md](firmware/README.en.md) | Build and flashing instructions, examples, pin assignments, and implementation notes |
| [firmware/Makefile](firmware/Makefile) | Example build checks and Wi-Fi example flash targets |

## Getting started

### Hardware

Use KiCad 9.0 or later. Some KiCad libraries are Git submodules; initialize them from the repository root:

```sh
git submodule update --init --recursive
```

Open the [KiCad project](hardware/tinygo-conf-2026.kicad_pro) and follow the [build guide](hardware/build/build.md) to assemble the badge.

### Firmware

Install TinyGo, connect the ESP32-S3-DevKit, and run:

```sh
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/blink
tinygo monitor --target esp32s3-box-3
```

Run `make smoketest` from `firmware/` to build all examples. See the [firmware README](firmware/README.en.md) for Wi-Fi and BLE setup and more examples.
