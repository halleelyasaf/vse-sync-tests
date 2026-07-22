// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"testing"
)

func TestParseDeviceJSONWrappedEmpty(t *testing.T) {
	entries, err := parseDeviceJSON([]byte(`{"device":[]}`))
	if err != nil {
		t.Fatalf("expected empty wrapped device response to parse, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(entries))
	}
}

func TestParsePinListJSONWrappedEmpty(t *testing.T) {
	pins, err := parsePinListJSON([]byte(`{"pin":[]}`))
	if err != nil {
		t.Fatalf("expected empty wrapped pin response to parse, got %v", err)
	}
	if len(pins) != 0 {
		t.Fatalf("expected 0 pins, got %d", len(pins))
	}
}

func TestParseDeviceJSONPlainArray(t *testing.T) {
	entries, err := parseDeviceJSON([]byte(`[{"id":1,"clock-id":42,"module-name":"ice","mode":"automatic","mode-supported":["automatic"],"lock-status":"locked","type":"eec"}]`))
	if err != nil {
		t.Fatalf("expected plain device array to parse, got %v", err)
	}
	if len(entries) != 1 || entries[0].ClockID != 42 {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestUnsupportedAttributeError(t *testing.T) {
	if err := unsupportedAttributeError(`{"ok":true}`); err != nil {
		t.Fatalf("expected nil for normal JSON, got %v", err)
	}
	err := unsupportedAttributeError(`error: attribute has no attribute with value 28`)
	if err == nil {
		t.Fatal("expected error for unsupported attribute marker")
	}
}
