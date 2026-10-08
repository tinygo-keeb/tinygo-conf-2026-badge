package main

import (
	"device/esp"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

// STORE3 survives badge.Reset and is unused by this TinyGo firmware. Avoid
// STORE0/STORE1 (ADC calibration) and STORE2 (exception diagnostics).
// RESET_REASON_CORE_SW = 3 is the reset caused by RTC_CNTL_SW_SYS_RST:
// https://github.com/espressif/esp-idf/blob/v5.1.2/components/soc/esp32s3/include/soc/reset_reasons.h
func takeMenuReturn() uint32 {
	marker := esp.RTC_CNTL.STORE3.Get()
	// Consume the marker once, so a later reset or power cycle shows TOP.
	esp.RTC_CNTL.STORE3.Set(0)
	if esp.RTC_CNTL.GetRESET_STATE_RESET_CAUSE_PROCPU() != 3 {
		return 0
	}
	return marker
}

func resetToMenu(selected int) {
	esp.RTC_CNTL.STORE3.Set(menuReturnMarker(selected))
	badge.Reset()
}
