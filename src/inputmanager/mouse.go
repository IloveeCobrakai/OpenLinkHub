package inputmanager

// Package: inputmanager
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/logger"
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	AbsMinX int32 = 0
	AbsMaxX int32 = 0
	AbsMinY int32 = 0
	AbsMaxY int32 = 0
)

// destroyVirtualMouse will destroy virtual mouse and close uinput device
func destroyVirtualMouse() {
	if virtualMousePointer != 0 {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiDevDestroy, 0); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual mouse")
		}

		if err := virtualMouseFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}

	if virtualMouseAbsPointer != 0 {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiDevDestroy, 0); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual absolute mouse")
		}

		if err := virtualMouseAbsFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}
}

func createVirtualMouse(vendorId, productId uint16) error {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	virtualMouseFile = f
	virtualMousePointer = f.Fd()

	uInputDevice := uInputUserDev{
		ID: inputID{
			BusType: 0x03, // BUS_USB
			Vendor:  vendorId,
			Product: productId,
			Version: 1,
		},
	}
	copy(uInputDevice.Name[:], "OpenLinkHub Virtual Mouse")

	// EV bits – kein evAbs hier
	for _, code := range []uint16{EvKey, EvSyn, EvRel} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetEvbit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to enable ev bit")
			return errno
		}
	}

	// REL axes
	for _, code := range []uint16{RelX, RelY, RelWheel, RelHWheel} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetRelbit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to enable rel bit")
			return errno
		}
	}

	// Buttons + modifier keys
	for _, code := range []uint16{btnLeft, btnRight, btnMiddle, btnBack, btnForward, keyLeftCtrl, keyRightCtrl, keyLeftAlt, keyLeftShift} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetKeybit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno, "code": code}).Error("Failed to enable key bit")
			return errno
		}
	}

	_, err = f.Write((*(*[unsafe.Sizeof(uInputDevice)]byte)(unsafe.Pointer(&uInputDevice)))[:])
	if err != nil {
		return err
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiDevCreate, 0); errno != 0 {
		return errno
	}
	return nil
}

// createVirtualMouseAbs will create a separate absolute-position virtual mouse device.
func createVirtualMouseAbs(vendorId, productId uint16) error {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0660)
	if err != nil {
		return err
	}

	virtualMouseAbsFile = f
	virtualMouseAbsPointer = f.Fd()

	uInputDevice := uInputUserDev{
		ID: inputID{
			BusType: 0x03, // BUS_USB
			Vendor:  vendorId,
			Product: productId,
			Version: 1,
		},
	}

	copy(uInputDevice.Name[:], "OpenLinkHub Virtual Absolute Mouse")

	bounds := common.GetScreenBounds()

	logger.Log(logger.Fields{
		"totalWidth":  bounds.TotalWidth,
		"totalHeight": bounds.TotalHeight,
	}).Info("Detected virtual desktop bounds")

	AbsMinX = 0
	AbsMaxX = bounds.TotalWidth - 1

	AbsMinY = 0
	AbsMaxY = bounds.TotalHeight - 1

	uInputDevice.AbsMin[AbsX] = AbsMinX
	uInputDevice.AbsMax[AbsX] = AbsMaxX
	uInputDevice.AbsMin[AbsY] = AbsMinY
	uInputDevice.AbsMax[AbsY] = AbsMaxY

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvKey)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse key events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvAbs)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse absolute events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvSyn)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse sync events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetAbsbit, uintptr(AbsX)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse ABS_X")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetAbsbit, uintptr(AbsY)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse ABS_Y")
		return errno
	}

	for _, code := range []uint16{btnLeft, btnRight, btnMiddle, btnBack, btnForward} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetKeybit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno, "code": code}).Error("Failed to enable absolute mouse button event")
			return errno
		}
	}

	if _, err := f.Write((*(*[unsafe.Sizeof(uInputDevice)]byte)(unsafe.Pointer(&uInputDevice)))[:]); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to write absolute mouse uinput struct")
		return err
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiDevCreate, 0); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to create absolute virtual mouse")
		return errno
	}

	return nil
}

func InputControlMouse(controlType uint16) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}
	actionType := getInputAction(controlType)
	if actionType == nil {
		return
	}
	for _, event := range createInputEvent(actionType.CommandCode, false) {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}

func InputControlMove(x, y int32) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}
	var events []inputEvent
	if x != 0 {
		events = append(events, inputEvent{Type: evRel, Code: RelX, Value: x})
	}
	if y != 0 {
		events = append(events, inputEvent{Type: evRel, Code: RelY, Value: y})
	}
	events = append(events, inputEvent{Type: evSyn, Code: 0, Value: 0})

	// Send events
	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}

// InputControlMoveAbsolute will move the virtual mouse to absolute coordinates.
func InputControlMoveAbsolute(x, y int32) {
	if virtualMouseAbsFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	if x < AbsMinX {
		x = AbsMinX
	}
	if x > AbsMaxX {
		x = AbsMaxX
	}
	if y < AbsMinY {
		y = AbsMinY
	}
	if y > AbsMaxY {
		y = AbsMaxY
	}

	if mouseStateX == x && mouseStateY == y {
		sendAbsEvents(x+1, y+1)
	}

	mouseStateX = x
	mouseStateY = y
	sendAbsEvents(x, y)
}

func sendAbsEvents(x, y int32) {
	events := []inputEvent{
		{
			Type:  evAbs,
			Code:  AbsX,
			Value: x,
		},
		{
			Type:  evAbs,
			Code:  AbsY,
			Value: y,
		},
		{
			Type:  evSyn,
			Code:  0,
			Value: 0,
		},
	}

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseAbsFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit absolute move event")
			return
		}
	}
}

// InputControlMoveAbsolutePixels will move mouse by X, Y with given screen size
func InputControlMoveAbsolutePixels(x, y, width, height int32) {
	if width <= 1 || height <= 1 {
		logger.Log(logger.Fields{"width": width, "height": height}).Error("Invalid screen size")
		return
	}

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= width {
		x = width - 1
	}
	if y >= height {
		y = height - 1
	}

	absX := x * AbsMaxX / (width - 1)
	absY := y * AbsMaxY / (height - 1)

	InputControlMoveAbsolute(absX, absY)
}

// InputControlScroll will trigger vertical scroll (up / down)
func InputControlScroll(up bool) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	value := int32(-1)
	if up {
		value = 1
	}

	scrollEvent := inputEvent{
		Type:  evRel,
		Code:  relWheel,
		Value: value,
	}

	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}
	events := []inputEvent{scrollEvent, syncEvent}

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit scroll event")
			return
		}
	}
}

// InputControlScrollHorizontal will trigger horizontal scroll (left / right)
func InputControlScrollHorizontal(left bool) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	value := int32(-1)
	if left {
		value = 1
	}

	scrollEvent := inputEvent{
		Type:  evRel,
		Code:  relWheel,
		Value: value,
	}

	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	events := []inputEvent{scrollEvent, syncEvent}

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit scroll event")
			return
		}
	}
}

// InputControlZoom will trigger zoom
func InputControlZoom(in bool) {
	if virtualMouseFile == nil {
		logger.Log(nil).Error("Virtual mouse is not present")
		return
	}

	scrollValue := int32(1)
	if !in {
		scrollValue = -1
	}

	pressCtrl := inputEvent{Type: evKey, Code: keyLeftCtrl, Value: 1}
	sync := inputEvent{Type: evSyn, Code: 0, Value: 0}
	scroll := inputEvent{Type: evRel, Code: relWheel, Value: scrollValue}
	releaseCtrl := inputEvent{Type: evKey, Code: keyLeftCtrl, Value: 0}

	events := []inputEvent{
		pressCtrl, sync,
		scroll, sync,
		releaseCtrl, sync,
	}

	for _, e := range events {
		if err := writeVirtualEvent(virtualMouseFile, &e); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit zoom event")
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// InputControlMouseHold will emulate input events based on virtual mouse and button hold.
func InputControlMouseHold(controlType uint16, press bool) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual keyboard is not present")
		return
	}
	var events []inputEvent

	// Get event key code
	actionType := getInputAction(controlType)
	if actionType == nil {
		return
	}

	// Create events
	events = createInputEventHold(actionType.CommandCode, press)

	// Send events
	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}

func GetVirtualMouse() *os.File {
	return virtualMouseFile
}
