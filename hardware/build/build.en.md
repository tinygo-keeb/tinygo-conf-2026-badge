## TinyGo Conf 2026 Badge Build Guide

[日本語](build.md)

This guide explains how to assemble the badge board distributed at TinyGo Conference 2026.
The required parts are listed below. Please provide your own soldering iron and other assembly tools.

### Parts

You will need the following parts:

- ESP32-S3 DevKit

https://akizukidenshi.com/catalog/g/g131148/

- Joystick

https://akizukidenshi.com/catalog/g/g115951/

- ST7789 display

https://akizukidenshi.com/catalog/g/g131019/

- One current-limiting resistor

https://akizukidenshi.com/catalog/g/g116332/

- Two key switches

https://shop.yushakobo.jp/collections/all-switches

- Two keycaps

https://shop.yushakobo.jp/collections/keycaps?sort_by=created-descending&filter.v.availability=1&filter.v.price.gte=&filter.v.price.lte=

- Two switch sockets

https://shop.yushakobo.jp/products/a01ps?variant=37665172521121

- MAX98357A

https://amzn.asia/d/054giKgU

- RGB LEDs (use two from the pack of five)

https://akizukidenshi.com/catalog/g/g115478/

- Infrared receiver module

https://akizukidenshi.com/catalog/g/g131157/

- Infrared LED

https://akizukidenshi.com/catalog/g/g112612/

- AHT21B temperature and humidity sensor

https://akizukidenshi.com/catalog/g/g130222/

- Two 20-pin socket strips

https://akizukidenshi.com/catalog/g/g103077/

- Two 2-pin socket strips

https://akizukidenshi.com/catalog/g/g110097/

### Assembly

#### LEDs / SK6812MINI-E

Start with the LEDs. Turn the board over and solder the surface-mount SK6812 LEDs to the back.

![](./img/board_bottom.jpg)

The SK6812 has a specific orientation. Look closely at the LED and you will see one corner cut at a 45-degree angle.

![](./img/led_zoom.jpg)

On the back of the board, a white L-shaped marking shows where each LED belongs.
Align the LED's cut corner with the white L. If either LED faces the wrong way, the LEDs will not light, so check their orientation before soldering.

Place both LEDs in the correct orientation, then solder them.

![](./img/led_1.jpg)

![](./img/led_2.jpg)

Tip: Pre-tinning the pads can make it easier to solder the LEDs, though it is not required.

![](./img/led_3.jpg)

Solder all four connections.

![](./img/led_4.jpg)

#### Switch sockets

Install the switch sockets. Remove them from their packaging and place them on the board according to the silkscreen.

![](./img/socket_1.jpg)

Apply the soldering iron from the side and feed solder into the joint. Check carefully that each socket is soldered to the board.

![](./img/socket_2.jpg)

Even if a joint looks soldered from above, a gap between the socket and board when viewed from the side means the solder has not reached the joint. Solder it again.

![](./img/socket_3.jpg)

#### Temperature and humidity sensor

Install the temperature and humidity sensor. First, solder its socket pins to the sensor module. If the pins are already soldered, insert the module from the front of the board.

![](./img/sensor_1.jpg)

After inserting the module, turn the board over and solder its pins.

![](./img/sensor_2.jpg)

Solder all four connections.

![](./img/sensor_3.jpg)

#### Infrared LED and resistor

Solder the infrared LED and resistor. Start with the resistor.

Bend the resistor leads at their bases into a U shape.

![](./img/r.jpg)

From the front of the board, insert the bent resistor into the two holes above and to the right of the Grove connector. The resistor has no polarity.

![](./img/r_1.jpg)

Turn the board over and solder the resistor.

![](./img/r_2.jpg)

![](./img/r_3.jpg)

Trim the excess leads with side cutters. Hold each lead with your free hand while cutting it so it does not fly toward your eyes or someone nearby.

![](./img/r_4.jpg)

Next, solder the infrared LED. It has a specific orientation: the longer lead goes on the left and the shorter lead on the right.

![](./img/ir_led_1.jpg)

Check the orientation and insert the LED from the front of the board.

![](./img/ir_led_2.jpg)

Turn the board over and solder it.

![](./img/ir_led_3.jpg)

Trim the excess leads with side cutters. Keep the trimmed leads; you will use them in a later step.

![](./img/ir_led_4.jpg)

#### Audio module

Install the audio module. First, solder the socket pins to the module. If they are already soldered, insert the module from the front of the board.

![](./img/max98357_0.jpg)

![](./img/max98357_1.jpg)

Turn the board over and solder the pins.

![](./img/max98357_2.jpg)

The pins on the back are now soldered.

![](./img/max98357_3.jpg)

Turn the board back over and solder the pin header on the left.

![](./img/max98357_4.jpg)

After soldering the pin headers on both sides, turn the board over and cut off all protruding, unused header pins.

![](./img/max98357_5.jpg)

![](./img/max98357_6.jpg)

Bend the lead saved from the resistor into a U shape.

![](./img/max98357_7.jpg)

Place it across the second and fourth pins from the top and solder it in place.

![](./img/max98357_8.jpg)

![](./img/max98357_9.jpg)

#### Grove connector

Insert the Grove connector from the front of the board.

![](./img/grove_0.jpg)

Turn the board over and solder the connector.

![](./img/grove_1.jpg)

Solder all four connections.

![](./img/grove_2.jpg)

#### Infrared receiver module

Install the infrared receiver module.

![](./img/ir_receive.jpg)

Insert it from the front of the board. The rounded, protruding receiver should face outward, away from the board.

![](./img/ir_receive_2.jpg)

Turn the board over and solder it.

![](./img/ir_receive_3.jpg)

![](./img/ir_receive_4.jpg)

Trim the excess leads with side cutters.

![](./img/ir_receive_5.jpg)

#### Speaker

Insert the speaker from the front of the board, with its + mark facing up.

![](./img/speaker_1.jpg)

Turn the board over and solder it.

![](./img/speaker_2.jpg)

#### Joystick

Insert the joystick from the front of the board.

![](./img/joystick_1.jpg)

![](./img/joystick_2.jpg)

Turn the board over and solder it.

![](./img/joystick_3.jpg)

![](./img/joystick_4.jpg)

#### ESP32-S3 DevKit

Solder the socket strips that will mount the ESP32-S3 DevKit to the board.

![](./img/devkit_1.jpg)

Fit the 20-pin and 2-pin socket strips onto the DevKit's pin headers.

![](./img/devkit_2.jpg)

Check from the side that the sockets are fully seated and level.

![](./img/devkit_3.jpg)

With the sockets fitted on both sides of the DevKit, insert the assembly from the back of the badge board.

![](./img/devkit_3_1.jpg)

With the DevKit inserted from the back, solder the socket pins from the front of the badge board.

![](./img/devkit_4.jpg)

Solder every pin.

![](./img/devkit_5.jpg)

#### Display

First, solder a pin header to the display.

![](./img/display_1.jpg)

Insert the header and display into a breadboard to hold them in place while soldering. If you do not have a breadboard, insert the header into the badge board and solder it there instead.

![](./img/display_2.jpg)

After soldering the header to the display, insert it from the front of the badge board.

Use tape to hold the display in place and keep it level. Check its position from the side. Once it is secure and level, turn the board over and solder it.

![](./img/display_3.jpg)

All soldering is complete.

#### Functional test

Check that the assembled parts work properly. Badges distributed at the workshop already have the `selftest` program flashed. Power the badge on and run the test.

If you assembled the badge yourself, flash the test program with:

```
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/selftest
```

If everything works, the display shows ALL OK as in the photo below.

![](./img/test_ok.jpg)

#### Case

Install the case. If you do not have one, print a case with a 3D printer.

Align the case, which is the same size as the board, with the holes at the board's corners.

![](./img/case_1.jpg)

![](./img/case_2.jpg)

Put the key switches into the case and insert them into the board from the front.

![](./img/sw_1.jpg)

![](./img/sw_2.jpg)

Finally, attach four clips at the corners so the board cannot come loose from the case.

![](./img/case_3.jpg)

The badge is complete. Great work!

![](./img/fin.jpg)

#### Optional expansion

You can fit a 2x8 socket strip into the holes at the bottom of the board. A mini breadboard fits in the blank name area in the center, making it easy to experiment with electronics.

https://akizukidenshi.com/catalog/g/g102761/
