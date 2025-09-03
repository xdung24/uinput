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
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// Device is a handle to an input device
type Device struct{ fd int }

func usage() {
	fmt.Fprintf(os.Stderr, "usage: uinputctl <command> [args...]\n")
	fmt.Fprintf(os.Stderr, "commands:\n")
	fmt.Fprintf(os.Stderr, "  device-key <device> <keycode> [down|up] - send key event to existing device\n")
	fmt.Fprintf(os.Stderr, "  device-touch <device> <x> <y>           - send touch event to existing device (raw device coordinates)\n")
	fmt.Fprintf(os.Stderr, "  screen-touch <device> <x> <y>           - send touch event to existing device (screen coordinates)\n")
	fmt.Fprintf(os.Stderr, "examples:\n")
	fmt.Fprintf(os.Stderr, "  uinputctl device-key /dev/input/event5 30 down\n")
	fmt.Fprintf(os.Stderr, "  uinputctl device-touch /dev/input/event5 2048 1394\n")
	fmt.Fprintf(os.Stderr, "  uinputctl screen-touch /dev/input/event5 40 40\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	command := flag.Arg(0)
	switch command {
	case "device-touch":
		if flag.NArg() < 4 {
			usage()
			os.Exit(2)
		}
		devicePath := flag.Arg(1)
		xStr := flag.Arg(2)
		yStr := flag.Arg(3)
		x, err := strconv.Atoi(xStr)
		if err != nil {
			log.Fatalf("invalid x coordinate: %v", err)
		}
		y, err := strconv.Atoi(yStr)
		if err != nil {
			log.Fatalf("invalid y coordinate: %v", err)
		}

		dev, err := OpenInputDevice(devicePath)
		if err != nil {
			log.Fatalf("open device: %v", err)
		}
		defer dev.CloseSimple()

		if err := dev.SendTouch(x, y); err != nil {
			log.Fatalf("send touch: %v", err)
		}
		fmt.Printf("touch sent to %s\n", devicePath)

	case "screen-touch":
		if flag.NArg() < 4 {
			usage()
			os.Exit(2)
		}
		devicePath := flag.Arg(1)
		xStr := flag.Arg(2)
		yStr := flag.Arg(3)
		screenX, err := strconv.Atoi(xStr)
		if err != nil {
			log.Fatalf("invalid x coordinate: %v", err)
		}
		screenY, err := strconv.Atoi(yStr)
		if err != nil {
			log.Fatalf("invalid y coordinate: %v", err)
		}

		// Transform screen coordinates to device coordinates using observed data
		deviceX := screenX * 60
		deviceY := screenY * 33
		fmt.Printf("Screen coordinates (%d, %d) -> Device coordinates (%d, %d)\n", screenX, screenY, deviceX, deviceY)

		dev, err := OpenInputDevice(devicePath)
		if err != nil {
			log.Fatalf("open device: %v", err)
		}
		defer dev.CloseSimple()

		if err := dev.SendTouch(deviceX, deviceY); err != nil {
			log.Fatalf("send touch: %v", err)
		}
		fmt.Printf("screen touch sent to %s\n", devicePath)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		usage()
		os.Exit(2)
	}
}

// OpenInputDevice opens an existing input device (like /dev/input/event5) for sending events
func OpenInputDevice(devicePath string) (*Device, error) {
	fd, err := unix.Open(devicePath, unix.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", devicePath, err)
	}
	return &Device{fd: fd}, nil
}

// Close destroys the virtual device and closes the underlying file descriptor.
func (d *Device) Close() error {
	if d == nil {
		return nil
	}
	cfd := C.int(d.fd)
	if C.do_ioctl(cfd, C.ulong(C.GO_UI_DEV_DESTROY), 0) < 0 {
		err := syscall.Errno(C.get_errno())
		// continue to close even if destroy failed
		_ = unix.Close(d.fd)
		return fmt.Errorf("ioctl UI_DEV_DESTROY failed: %v", err)
	}
	return unix.Close(d.fd)
}

// CloseSimple closes the file descriptor without destroying uinput device (for direct device access)
func (d *Device) CloseSimple() error {
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
