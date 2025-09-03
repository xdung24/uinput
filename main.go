//go:build linux

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

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
		defer dev.Close()

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
		defer dev.Close()

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
