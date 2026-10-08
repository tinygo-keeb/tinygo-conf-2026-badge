# Hardware (tinygo-conf-2026)

[日本語](README.md)

This KiCad project defines the development board for TinyGo Conference 2026. It brings the display, audio, and input devices together on one board around an ESP32-S3-DevKit.

## Main components and features

- **ESP32-S3-DevKit** — main board mounted through pin sockets
- **ST7789 display** — SPI connection (CS / DC / RES / SCL / DIN), 8-pin connector
- **MAX98357 (Adafruit)** — I2S audio amplifier (BCLK / LRC / DIN) and speaker
- **Two RGB LEDs** — connected in series
- **Analog joystick (ALPS RKJXV122400R)** — X/Y axes and push button
- **Two push switches**
- **Infrared LED and receiver** — IR_LED / IR_DATA
- **4-pin I2C connector** — for external modules, with a Grove-compatible footprint
- **AHT21B** — temperature and humidity sensor

## Files

| Path | Description |
| --- | --- |
| `tinygo-conf-2026.kicad_pro` | KiCad project file |
| `tinygo-conf-2026.kicad_sch` | Schematic |
| `tinygo-conf-2026.kicad_pcb` | PCB layout |
| `tinygo-conf-2026-devkit/` | Manufacturing Gerber and drill files |
| `lib/` | Symbol and footprint libraries, including the submodules listed below |
| `lib/sglib.kicad_sym`, `lib/sglib.pretty/` | Custom libraries for the joystick, Grove connector, and other parts |
| `fp-lib-table`, `sym-lib-table` | Project library tables |

## Requirements

- KiCad 9.0 or later

## Setup

Some libraries are managed as Git submodules. Run this command from the repository root:

```sh
git submodule update --init --recursive
```

Then open `hardware/tinygo-conf-2026.kicad_pro` in KiCad.

### Submodules

| Path | Purpose | Source |
| --- | --- | --- |
| `lib/espressif` | ESP32 symbols | espressif/kicad-libraries |
| `lib/kbd` | ESP32-S3-DevKit symbols and more | foostan/kbd |
| `lib/MAX98357` | MAX98357 amplifier | besi/kicad-adafruit-MAX98357 |
| `lib/st7789` | ST7789 display | BennyLuca/Kicad_Components_Library |
| `lib/sparkfun` | SparkFun library | sparkfun/SparkFun-KiCad-Libraries |

## Manufacturing files

The `tinygo-conf-2026-devkit/` directory contains the complete two-layer board output: front and back copper, solder mask, paste, silkscreen, board outline, and PTH/NPTH drill files.

## Assembly

See the [build guide](build/build.en.md) for the parts list and illustrated instructions.
