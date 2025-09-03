//go:build linux

package main

/*
#cgo CFLAGS: -D_GNU_SOURCE
#include <linux/uinput.h>
#include <linux/input.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#include <sys/time.h>
#include <stdlib.h>
#include <errno.h>

static int emit_event(int fd, unsigned short type, unsigned short code, int value) {
    struct input_event ev;
    memset(&ev, 0, sizeof(ev));
    gettimeofday(&ev.time, NULL);
    ev.type = type;
    ev.code = code;
    ev.value = value;
    return write(fd, &ev, sizeof(ev));
}

static int get_errno() {
    return errno;
}

static int do_ioctl(int fd, unsigned long request, unsigned long arg) {
    return ioctl(fd, request, arg);
}

// Constants for Go access
const int GO_EV_KEY = EV_KEY;
const int GO_EV_ABS = EV_ABS;
const int GO_EV_SYN = EV_SYN;
const int GO_SYN_REPORT = SYN_REPORT;
const int GO_SYN_MT_REPORT = SYN_MT_REPORT;
const int GO_UI_SET_EVBIT = UI_SET_EVBIT;
const int GO_UI_SET_KEYBIT = UI_SET_KEYBIT;
const int GO_UI_SET_ABSBIT = UI_SET_ABSBIT;
const int GO_UI_DEV_CREATE = UI_DEV_CREATE;
const int GO_UI_DEV_DESTROY = UI_DEV_DESTROY;
const int GO_BUS_USB = BUS_USB;

// Touch event constants
const int GO_BTN_TOUCH = BTN_TOUCH;
const int GO_ABS_MT_TRACKING_ID = ABS_MT_TRACKING_ID;
const int GO_ABS_MT_POSITION_X = ABS_MT_POSITION_X;
const int GO_ABS_MT_POSITION_Y = ABS_MT_POSITION_Y;

*/
import "C"

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// Device is a handle to an input device
type Device struct{ fd int }

// OpenInputDevice opens an existing input device (like /dev/input/event5) for sending events
func OpenInputDevice(devicePath string) (*Device, error) {
	fd, err := unix.Open(devicePath, unix.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", devicePath, err)
	}
	return &Device{fd: fd}, nil
}

// Close closes the file descriptor without destroying uinput device (for direct device access)
func (d *Device) Close() error {
	if d == nil {
		return nil
	}
	return unix.Close(d.fd)
}

// SendTouch sends a complete touch event sequence (touch down at coordinates and release)
// This exactly mimics the working sendevent sequence
func (d *Device) SendTouch(x, y int) error {
	if d == nil {
		return fmt.Errorf("device is nil")
	}
	cfd := C.int(d.fd)

	// Exact sequence from working sendevent commands:
	// sendevent /dev/input/event5 1 330 1
	if C.emit_event(cfd, C.ushort(C.GO_EV_KEY), C.__u16(C.GO_BTN_TOUCH), 1) < 0 {
		return fmt.Errorf("emit BTN_TOUCH failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 3 57 0
	if C.emit_event(cfd, C.ushort(C.GO_EV_ABS), C.__u16(C.GO_ABS_MT_TRACKING_ID), 0) < 0 {
		return fmt.Errorf("emit MT_TRACKING_ID failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 3 53 1724  (using x parameter instead of hardcoded 1724)
	if C.emit_event(cfd, C.ushort(C.GO_EV_ABS), C.__u16(C.GO_ABS_MT_POSITION_X), C.int(x)) < 0 {
		return fmt.Errorf("emit MT_POSITION_X failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 3 54 1316  (using y parameter instead of hardcoded 1316)
	if C.emit_event(cfd, C.ushort(C.GO_EV_ABS), C.__u16(C.GO_ABS_MT_POSITION_Y), C.int(y)) < 0 {
		return fmt.Errorf("emit MT_POSITION_Y failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 0 2 0
	if C.emit_event(cfd, C.ushort(C.GO_EV_SYN), C.ushort(C.GO_SYN_MT_REPORT), 0) < 0 {
		return fmt.Errorf("emit SYN_MT_REPORT failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 0 0 0
	if C.emit_event(cfd, C.ushort(C.GO_EV_SYN), C.ushort(C.GO_SYN_REPORT), 0) < 0 {
		return fmt.Errorf("emit SYN_REPORT failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 0 2 0
	if C.emit_event(cfd, C.ushort(C.GO_EV_SYN), C.ushort(C.GO_SYN_MT_REPORT), 0) < 0 {
		return fmt.Errorf("emit SYN_MT_REPORT failed: %v", syscall.Errno(C.get_errno()))
	}

	// sendevent /dev/input/event5 0 0 0
	if C.emit_event(cfd, C.ushort(C.GO_EV_SYN), C.ushort(C.GO_SYN_REPORT), 0) < 0 {
		return fmt.Errorf("emit SYN_REPORT failed: %v", syscall.Errno(C.get_errno()))
	}

	return nil
}
