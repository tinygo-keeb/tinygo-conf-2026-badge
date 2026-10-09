package main

import "time"

const (
	menuRows         = 8
	directionPress   = 500
	directionRelease = 350
	buttonDebounce   = 25 * time.Millisecond
	menuRepeatFirst  = 400 * time.Millisecond
	menuRepeatNext   = 120 * time.Millisecond
	menuReturnHold   = time.Second
	menuReturnMagic  = uint32(0x414c4c00) // "ALL" followed by the selected index.
)

type buttonTransition struct {
	raw, stable bool
	changedAt   time.Time
}

type menuReturnGesture struct {
	startedAt time.Time
	fired     bool
}

// Require both switches continuously for one second. Ordinary single presses,
// brief chords and joystick operations never trigger a return.
func (g *menuReturnGesture) update(now time.Time, sw1, sw2 bool) bool {
	if !sw1 || !sw2 {
		g.startedAt, g.fired = time.Time{}, false
		return false
	}
	if g.startedAt.IsZero() {
		g.startedAt = now
	}
	if !g.fired && now.Sub(g.startedAt) >= menuReturnHold {
		g.fired = true
		return true
	}
	return false
}

func (b *buttonTransition) pressed(now time.Time, raw bool) bool {
	if raw != b.raw {
		b.raw = raw
		b.changedAt = now
	}
	if b.stable != raw && now.Sub(b.changedAt) >= buttonDebounce {
		b.stable = raw
		return raw
	}
	return false
}

type navigation struct {
	menu           bool
	selected       int
	first          int
	sw1, joy       buttonTransition
	direction      int
	repeatAt       time.Time
	waitForNeutral bool
}

func menuReturnMarker(selected int) uint32 {
	if selected < 0 || selected > 255 {
		return 0
	}
	return menuReturnMagic | uint32(selected)
}

func navigationFromMenuReturn(marker uint32, count int) navigation {
	selected := int(marker & 0xff)
	if marker&0xffffff00 != menuReturnMagic || selected >= count {
		return navigation{}
	}
	return navigation{
		menu:           true,
		selected:       selected,
		first:          max(0, selected-menuRows+1),
		waitForNeutral: true,
	}
}

// update returns whether the screen changed and whether to launch an example.
// Sampling both buttons on every screen prevents a held SW1 from entering the
// menu and launching the first example on the same press.
func (n *navigation) update(now time.Time, x, y int, sw1, sw2, joy bool, count int) (changed, launch bool) {
	enter := n.sw1.pressed(now, sw1)
	start := n.joy.pressed(now, joy)
	if n.waitForNeutral {
		// Consume held controls from the running example until all are released.
		if x > -directionRelease && x < directionRelease && y > -directionRelease && y < directionRelease && !sw1 && !sw2 && !joy && !n.sw1.stable && !n.joy.stable {
			n.waitForNeutral = false
		}
		return false, false
	}
	if !n.menu {
		if enter {
			n.menu = true
			n.direction = 0
			return true, false
		}
		return false, false
	}
	if x < -directionPress {
		n.menu = false
		n.direction = 0
		return true, false
	}
	if count == 0 {
		return false, false
	}
	if enter || start {
		return false, true
	}

	dir := 0
	if y > directionPress || (n.direction == -1 && y > directionRelease) {
		dir = -1 // Up selects the preceding item.
	} else if y < -directionPress || (n.direction == 1 && y < -directionRelease) {
		dir = 1
	}
	move := false
	if dir != n.direction {
		n.direction = dir
		if dir != 0 {
			move = true
			n.repeatAt = now.Add(menuRepeatFirst)
		}
	} else if dir != 0 && !now.Before(n.repeatAt) {
		move = true
		n.repeatAt = now.Add(menuRepeatNext)
	}
	if !move {
		return false, false
	}
	n.selected = (n.selected + dir + count) % count
	if n.selected < n.first {
		n.first = n.selected
	} else if n.selected >= n.first+menuRows {
		n.first = n.selected - menuRows + 1
	}
	return true, false
}
