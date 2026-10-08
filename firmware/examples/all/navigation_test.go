package main

import (
	"testing"
	"time"
)

const testProgramCount = 20

func TestSW1OpensMenuWithoutLaunchingOnHeldPress(t *testing.T) {
	n := navigation{}
	now := time.Unix(1, 0)
	changed, launch := n.update(now, 0, 0, true, false, testProgramCount)
	if changed || launch {
		t.Fatal("button bounce changed the screen")
	}
	changed, launch = n.update(now.Add(30*time.Millisecond), 0, 0, true, false, testProgramCount)
	if !changed || launch || !n.menu {
		t.Fatal("first SW1 press must only open the menu")
	}
	for tick := 40; tick <= 500; tick += 10 {
		_, launch = n.update(now.Add(time.Duration(tick)*time.Millisecond), 0, 0, true, false, testProgramCount)
		if launch {
			t.Fatal("holding the menu-opening button launched an example")
		}
	}
	n.update(now.Add(510*time.Millisecond), 0, 0, false, false, testProgramCount)
	n.update(now.Add(540*time.Millisecond), 0, 0, false, false, testProgramCount)
	n.update(now.Add(550*time.Millisecond), 0, 0, true, false, testProgramCount)
	_, launch = n.update(now.Add(580*time.Millisecond), 0, 0, true, false, testProgramCount)
	if !launch {
		t.Fatal("second SW1 press did not launch the selected example")
	}
}

func TestAllExamplesCanBeSelectedWithScrollingAndWrap(t *testing.T) {
	n := navigation{menu: true}
	now := time.Unix(1, 0)
	for index := 1; index <= testProgramCount; index++ {
		changed, launch := n.update(now, 0, -800, false, false, testProgramCount)
		if !changed || launch || n.selected != index%testProgramCount {
			t.Fatalf("down step %d: selected %d", index, n.selected)
		}
		if n.selected < n.first || n.selected >= n.first+menuRows {
			t.Fatalf("selected item %d is outside the visible menu", n.selected)
		}
		now = now.Add(20 * time.Millisecond)
		n.update(now, 0, 0, false, false, testProgramCount)
		now = now.Add(20 * time.Millisecond)
	}
	n.update(now, 0, 800, false, false, testProgramCount)
	if n.selected != testProgramCount-1 || n.first != testProgramCount-menuRows {
		t.Fatal("up from the first item did not wrap to the last page")
	}
}

func TestHeldDirectionRepeatsAndIgnoresThresholdNoise(t *testing.T) {
	n := navigation{menu: true}
	now := time.Unix(1, 0)
	n.update(now, 0, -800, false, false, testProgramCount)
	n.update(now.Add(399*time.Millisecond), 0, -400, false, false, testProgramCount)
	if n.selected != 1 {
		t.Fatal("direction repeated too early or noise triggered a new step")
	}
	n.update(now.Add(400*time.Millisecond), 0, -400, false, false, testProgramCount)
	if n.selected != 2 {
		t.Fatal("held direction did not repeat after its initial delay")
	}
	n.update(now.Add(520*time.Millisecond), 0, -400, false, false, testProgramCount)
	if n.selected != 3 {
		t.Fatal("held direction did not repeat at the repeat interval")
	}
	n.update(now.Add(530*time.Millisecond), 0, -300, false, false, testProgramCount)
	n.update(now.Add(900*time.Millisecond), 0, -400, false, false, testProgramCount)
	if n.selected != 3 {
		t.Fatal("released direction repeated in the dead zone")
	}
	n.update(now.Add(910*time.Millisecond), 0, 800, false, false, testProgramCount)
	if n.selected != 2 {
		t.Fatal("changing to up did not immediately select the previous item")
	}
}

func TestLeftReturnsToLogoAndTakesPriorityOverStart(t *testing.T) {
	n := navigation{menu: true, selected: 12, first: 5}
	now := time.Unix(1, 0)
	n.update(now, 0, 0, true, true, testProgramCount)
	changed, launch := n.update(now.Add(30*time.Millisecond), -800, 800, true, true, testProgramCount)
	if !changed || launch || n.menu || n.selected != 12 {
		t.Fatal("left must return to TOP, even if start or up is also held")
	}
	n.update(now.Add(100*time.Millisecond), 0, 0, true, true, testProgramCount)
	if n.menu {
		t.Fatal("held start button reopened the menu after returning to TOP")
	}
}

func TestJoystickPressStartsOnlyFromMenu(t *testing.T) {
	n := navigation{}
	now := time.Unix(1, 0)
	n.update(now, 0, 0, false, true, testProgramCount)
	changed, launch := n.update(now.Add(30*time.Millisecond), 0, 0, false, true, testProgramCount)
	if changed || launch || n.menu {
		t.Fatal("joystick press must not leave the logo screen")
	}
	n = navigation{menu: true, selected: 8}
	n.update(now, 0, 0, false, true, testProgramCount)
	_, launch = n.update(now.Add(30*time.Millisecond), 0, 0, false, true, testProgramCount)
	if !launch || n.selected != 8 {
		t.Fatal("joystick press did not start the selected example")
	}
}

func TestMenuReturnRestoresEverySelection(t *testing.T) {
	for selected := 0; selected < testProgramCount; selected++ {
		n := navigationFromMenuReturn(menuReturnMarker(selected), testProgramCount)
		if !n.menu || n.selected != selected || !n.waitForNeutral {
			t.Fatalf("return from example %d did not restore its menu selection", selected)
		}
		if selected < n.first || selected >= n.first+menuRows {
			t.Fatalf("restored selection %d is not visible", selected)
		}
	}
}

func TestInvalidMenuReturnStartsAtLogo(t *testing.T) {
	markers := []uint32{0, 0xffffffff, 0x55550000, menuReturnMagic ^ 0x10000, menuReturnMarker(testProgramCount), menuReturnMarker(-1), menuReturnMarker(256)}
	for _, marker := range markers {
		if n := navigationFromMenuReturn(marker, testProgramCount); n.menu || n.selected != 0 {
			t.Fatalf("invalid marker %#x opened the menu", marker)
		}
	}
}

func TestRestoredMenuWaitsForStickAndButtonsToBeReleased(t *testing.T) {
	n := navigationFromMenuReturn(menuReturnMarker(12), testProgramCount)
	now := time.Unix(1, 0)
	for tick := 0; tick <= 100; tick += 10 {
		changed, launch := n.update(now.Add(time.Duration(tick)*time.Millisecond), -800, 800, true, true, testProgramCount)
		if changed || launch || !n.menu || n.selected != 12 || !n.waitForNeutral {
			t.Fatal("held controls left the restored menu or started an example")
		}
	}
	n.update(now.Add(110*time.Millisecond), 0, 0, false, false, testProgramCount)
	n.update(now.Add(140*time.Millisecond), 0, 0, false, false, testProgramCount)
	if n.waitForNeutral {
		t.Fatal("restored menu did not become active after the controls were released")
	}
	changed, launch := n.update(now.Add(150*time.Millisecond), -800, 0, false, false, testProgramCount)
	if !changed || launch || n.menu {
		t.Fatal("a new left tilt in the selection menu must still return to TOP")
	}
}
