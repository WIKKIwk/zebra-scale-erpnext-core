//go:build !linux

package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type ZebraPrinter struct {
	DevicePath   string
	VendorID     string
	ProductID    string
	Manufacturer string
	Product      string
	Serial       string
	BusNum       string
	DevNum       string
}

func (p ZebraPrinter) IsZebra() bool {
	text := strings.ToLower(strings.TrimSpace(p.Manufacturer + " " + p.Product))
	return strings.Contains(text, "zebra") || strings.Contains(text, "ztc")
}

func (p ZebraPrinter) DisplayName() string {
	name := strings.TrimSpace(p.Manufacturer + " " + p.Product)
	if name == "" {
		return "unsupported"
	}
	return name
}

func FindZebraPrinters() ([]ZebraPrinter, error) {
	return nil, errors.New("zebra: unsupported on this platform")
}

func SelectZebraPrinter(preferred string) (ZebraPrinter, error) {
	return ZebraPrinter{}, fmt.Errorf("zebra: unsupported on %s", runtime.GOOS)
}

func zebraSendRaw(device string, payload []byte) error {
	return fmt.Errorf("zebra: raw transport unsupported on %s", runtime.GOOS)
}

func zebraSendSGD(device string, command string) error {
	return fmt.Errorf("zebra: SGD transport unsupported on %s", runtime.GOOS)
}

func queryZebraHostStatus(device string, timeout time.Duration) (string, error) {
	return "", fmt.Errorf("zebra: host status unsupported on %s", runtime.GOOS)
}

func queryZebraSGDVar(device, key string, timeout time.Duration) (string, error) {
	return "", fmt.Errorf("zebra: query unsupported on %s", runtime.GOOS)
}
