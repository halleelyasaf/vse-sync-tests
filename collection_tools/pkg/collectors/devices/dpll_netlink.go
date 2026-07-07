// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/callbacks"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/fetcher"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

var states = map[string]string{
	"unknown":       "-1",
	"invalid":       "0",
	"freerun":       "1",
	"locked":        "2",
	"locked-ho-acq": "3",
	"holdover":      "4",
}

const (
	OnePPSLabel = "GNSS-1PPS"
	SMA1Label   = "SMA1"
	SMA2Label   = "SMA2"

	OnePPSSubtype  = "dpll"
	SMA1Subtype    = "dpll-sma1"
	UnknownSubtype = "unknown"

	InputDirection = "input"
	ConnectedState = "connected"

	EECOffsetParentID      = 0
	PPSOffesetParentID     = 1
	DPLLPhaseOffsetDivider = 1000
)

type DevNetlinkDPLLInfo struct {
	PinType   string
	Timestamp string `fetcherKey:"date"       json:"timestamp"`
	EECState  string `fetcherKey:"eec"        json:"eecstate"`
	PPSState  string `fetcherKey:"pps"        json:"state"`
	PPSOffset int64  `fetcherKey:"pps_offset" json:"terror"`
	EECOffset int64  `fetcherKey:"eec_offset" json:"eecterror"`
}

func convertNetlinkOffset(offset int64) float64 {
	// Convert to nano seconds with 3 decimal places
	return float64(int64(math.Round(float64(offset/DPLLPhaseOffsetDivider)))) / 1000 //nolint:mnd // this is just for decimal places
}

// AnalyserJSON returns the json expected by the analysers
func (dpllInfo *DevNetlinkDPLLInfo) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	subType := UnknownSubtype

	switch dpllInfo.PinType {
	case OnePPSLabel:
		subType = OnePPSSubtype
	case SMA1Label:
		subType = SMA1Subtype
	default:
		// For other pins (timing cards, fallback pins), use a generic "dpll" subtype
		// instead of "unknown" to indicate valid DPLL data from a non-standard source
		if dpllInfo.PinType != "" {
			subType = OnePPSSubtype // Use "dpll" prefix for compatibility
		}
	}

	formatted := callbacks.AnalyserFormatType{
		ID: subType + "/time-error",
		Data: map[string]any{
			"timestamp": dpllInfo.Timestamp,
			"eecstate":  dpllInfo.EECState,
			"state":     dpllInfo.PPSState,
			"terror":    convertNetlinkOffset(dpllInfo.PPSOffset),
			"eecterror": convertNetlinkOffset(dpllInfo.EECOffset),
		},
	}

	return []*callbacks.AnalyserFormatType{&formatted}, nil
}

type NetlinkStateEntry struct {
	LockStatus string `json:"lock-status"` //nolint:tagliatelle // not my choice
	Driver     string `json:"module-name"` //nolint:tagliatelle // not my choice
	ClockType  string `json:"type"`        //nolint:tagliatelle // not my choice
	ClockID    uint64 `json:"clock-id"`    //nolint:tagliatelle // not my choice
	ID         int    `json:"id"`          //nolint:tagliatelle // not my choice
}

// # Example output
// [{'clock-id': 5799633565435100136,
//   'id': 0,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'eec'},
//  {'clock-id': 5799633565435100136,
//   'id': 1,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'pps'}]

type NetlinkPin struct {
	Type                 string                            `json:"type"`                //nolint:tagliatelle // not my choice
	ModuleName           string                            `json:"module-name"`         //nolint:tagliatelle // not my choice
	Label                string                            `json:"board-label"`         //nolint:tagliatelle // not my choice
	Capabilities         []string                          `json:"capabilities"`        //nolint:tagliatelle // not my choice
	FrequenciesSupported []*NetlinkFrequencySupportedRange `json:"frequency-supported"` //nolint:tagliatelle // not my choice
	ParentDevices        []*NetlinkParentDevice            `json:"parent-device"`       //nolint:tagliatelle // not my choice
	ParentPins           []*NetlinkParentPin               `json:"parent-pin"`          //nolint:tagliatelle // not my choice
	ClockID              uint64                            `json:"clock-id"`            //nolint:tagliatelle // not my choice
	Frequency            uint64                            `json:"frequency"`           //nolint:tagliatelle // not my choice
	ID                   int32                             `json:"id"`                  //nolint:tagliatelle // not my choice
	PhaseAdjust          int32                             `json:"phase-adjust"`        //nolint:tagliatelle // not my choice
	PhaseAdjustMax       int32                             `json:"phase-adjust-max"`    //nolint:tagliatelle // not my choice
	PhaseAdjustMin       int32                             `json:"phase-adjust-min"`    //nolint:tagliatelle // not my choice
}

type NetlinkParentDevice struct {
	Direction   string `json:"direction"`    //nolint:tagliatelle // not my choice
	State       string `json:"state"`        //nolint:tagliatelle // not my choice
	ParentID    int    `json:"parent-id"`    //nolint:tagliatelle // not my choice
	PhaseOffset int64  `json:"phase-offset"` //nolint:tagliatelle // not my choice
	Prio        int    `json:"prio"`         //nolint:tagliatelle // not my choice
}

type NetlinkParentPin struct {
	State    string `json:"state"`     //nolint:tagliatelle // not my choice
	ParentID int32  `json:"parent-id"` //nolint:tagliatelle // not my choice
}

type NetlinkFrequencySupportedRange struct {
	Max int32 `json:"frequency-max"` //nolint:tagliatelle // not my choice
	Min int32 `json:"frequency-min"` //nolint:tagliatelle // not my choice
}

// # Example output
// {
// 	'board-label': 'GNSS-1PPS',
// 	'capabilities': 6,
// 	'clock-id': 5799633565433967608,
// 	'frequency': 1,
// 	'frequency-supported': [
// 		{
// 			'frequency-max': 1,
// 			'frequency-min': 1
// 		}
// 	],
// 	'id': 6,
// 	'module-name': 'ice',
// 	'parent-device': [
// 		{
// 			'direction': 'input',
// 			'parent-id': 0,
// 			'phase-offset': 406616064733390,
// 			'prio': 0,
// 			'state': 'connected'
// 		},
// 		{
// 			'direction': 'input',
// 			'parent-id': 1,
// 			'phase-offset': -1870360,
// 			'prio': 0,
// 			'state': 'connected'
// 		}
// 	],
// 	'phase-adjust': 0,
// 	'phase-adjust-max': 16723,
// 	'phase-adjust-min': -16723,
// 	'type': 'gnss'
// },

// Wrapper structs for iproute2 dpll -j output which wraps arrays in
// {"device": [...]} or {"pin": [...]}. We fall back to plain arrays
// for backward compatibility with the old YNL cli.py format.
type dpllDeviceResponse struct {
	Device []NetlinkStateEntry `json:"device"`
}

type dpllPinListResponse struct {
	Pin []*NetlinkPin `json:"pin"`
}

type dpllPinSingleResponse struct {
	Pin []NetlinkPin `json:"pin"`
}

// parseDeviceJSON tries the wrapped {"device": [...]} format first,
// then falls back to a plain [...] array.
func parseDeviceJSON(raw []byte) ([]NetlinkStateEntry, error) {
	var wrapped dpllDeviceResponse
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Device) > 0 {
		return wrapped.Device, nil
	}

	var entries []NetlinkStateEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device JSON: %w", err)
	}
	return entries, nil
}

// parsePinListJSON tries the wrapped {"pin": [...]} format first,
// then falls back to a plain [...] array.
func parsePinListJSON(raw []byte) ([]*NetlinkPin, error) {
	var wrapped dpllPinListResponse
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Pin) > 0 {
		return wrapped.Pin, nil
	}

	var entries []*NetlinkPin
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pin list JSON: %w", err)
	}
	return entries, nil
}

// parseSinglePinJSON tries the wrapped {"pin": [{...}]} format first,
// then falls back to a plain {...} object.
func parseSinglePinJSON(raw []byte) (NetlinkPin, error) {
	var wrapped dpllPinSingleResponse
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Pin) > 0 {
		return wrapped.Pin[0], nil
	}

	var pin NetlinkPin
	if err := json.Unmarshal(raw, &pin); err != nil {
		return pin, fmt.Errorf("failed to unmarshal single pin JSON: %w", err)
	}
	return pin, nil
}

var (
	dpllNetlinkFetcher map[uint64]*fetcher.Fetcher
	dpllClockIDFetcher map[string]*fetcher.Fetcher
)

func init() {
	dpllNetlinkFetcher = make(map[uint64]*fetcher.Fetcher)
	dpllClockIDFetcher = make(map[string]*fetcher.Fetcher)
}

func buildPostProcessDPLLNetlink(clockID uint64) fetcher.PostProcessFuncType {
	return func(result map[string]string) (map[string]any, error) {
		processedResult := make(map[string]any)

		deviceJSON := result["dpll-netlink-device"]
		if strings.Contains(deviceJSON, "has no attribute with value") {
			return processedResult, fmt.Errorf(
				"dpll tool does not recognise a netlink attribute — "+
					"update the container image (NETLINK_DEBUG_CONTAINER_IMAGE env var): %s",
				deviceJSON,
			)
		}

		entries, err := parseDeviceJSON([]byte(deviceJSON))
		if err != nil {
			log.Errorf("Failed to unmarshal netlink device output: %s", err.Error())
		}

		log.Debug("entries: ", entries)

		for _, entry := range entries {
			if entry.ClockID == clockID {
				state, ok := states[entry.LockStatus]
				if !ok {
					log.Errorf("Unknown state: %s", state)
					state = "-1"
				}

				processedResult[entry.ClockType] = state
			}
		}

		pin, err := parseSinglePinJSON([]byte(result["dpll-netlink-offset"]))
		if err != nil {
			log.Errorf("Failed to unmarshal netlink pin output: %s", err.Error())
		}

		for _, parentPin := range pin.ParentDevices {
			switch parentPin.ParentID % 2 {
			case EECOffsetParentID:
				processedResult["ecc_offset"] = parentPin.PhaseOffset
			case PPSOffesetParentID:
				processedResult["pps_offset"] = parentPin.PhaseOffset
			}
		}

		return processedResult, nil
	}
}

// BuildDPLLNetlinkDeviceFetcher popluates the fetcher required for
// collecting the DPLLInfo
func BuildDPLLNetlinkDeviceFetcher(params NetlinkParameters) error { //nolint:dupl // Further dedup risks be too abstract or fragile
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key:     "dpll-netlink-device",
				Command: "dpll -j device show",
				Trim:    true,
			},
			{
				Key:     "dpll-netlink-offset",
				Command: fmt.Sprintf("dpll -j pin show id %d", params.OffsetPin),
				Trim:    true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll netlink: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll netlink: %w", err)
	}

	dpllNetlinkFetcher[params.ClockID] = fetcherInst
	fetcherInst.SetPostProcessor(buildPostProcessDPLLNetlink(params.ClockID))

	return nil
}

// GetDevDPLLInfo returns the device DPLL info for an interface.
func GetDevDPLLNetlinkInfo(ctx clients.ExecContext, params NetlinkParameters) (*DevNetlinkDPLLInfo, error) {
	dpllInfo := &DevNetlinkDPLLInfo{PinType: params.PinType}

	fetcherInst, fetchedInstanceOk := dpllNetlinkFetcher[params.ClockID]
	if !fetchedInstanceOk {
		err := BuildDPLLNetlinkDeviceFetcher(params)
		if err != nil {
			return dpllInfo, err
		}

		fetcherInst, fetchedInstanceOk = dpllNetlinkFetcher[params.ClockID]
		if !fetchedInstanceOk {
			return dpllInfo, errors.New("failed to create fetcher for DPLLInfo using netlink interface")
		}
	}

	err := fetcherInst.Fetch(ctx, dpllInfo)
	if err != nil {
		return dpllInfo, fmt.Errorf("failed to fetch dpllInfo via netlink: %w", err)
	}

	return dpllInfo, nil
}

func BuildNetlinkInfoFetcher(interfaceName string) error {
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key: "dpll-netlink-clock-serial-number",
				Command: fmt.Sprintf(
					`export IFNAME=%s; export BUSID=$(readlink /sys/class/net/$IFNAME/device | xargs basename | cut -d ':' -f 2,3);`+
						` echo $(lspci -v | grep $BUSID -A 20 |grep 'Serial Number' | awk '{print $NF}' | tr -d '-')`,
					interfaceName,
				),
				Trim: true,
			},
			{
				Key:     "dpll-netlink-pins",
				Command: "dpll -j pin show",
				Trim:    true,
			},
			{
				Key:     "dpll-netlink-devices",
				Command: "dpll -j device show",
				Trim:    true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll clock ID: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll clock ID: %w", err)
	}

	fetcherInst.SetPostProcessor(postProcessDPLLNetlinkClockID)
	dpllClockIDFetcher[interfaceName] = fetcherInst

	return nil
}

func selectPin(pinsJSON []byte, clockID uint64) (int32, string, error) { //nolint:funlen,gocritic,cyclop // allow slightly longer function for sake of readability
	entries, err := parsePinListJSON(pinsJSON)
	if err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal netlink output: %w", err)
	}

	if len(entries) == 0 {
		return 0, "", utils.NewRequirementsNotMetError(errors.New("no pins found"))
	}

	var OnePPSPin, SMA1Pin *NetlinkPin

	log.Debugf("selectPin: looking for clockID %d among %d pins", clockID, len(entries))

	matchingPinCount := 0
	for _, pin := range entries {
		if pin.ClockID != clockID {
			log.Debugf("selectPin: skipping pin ID=%d label=%s (clockID %d != %d)", pin.ID, pin.Label, pin.ClockID, clockID)
			continue
		}
		matchingPinCount++
		log.Debugf("selectPin: found matching pin ID=%d label=%s type=%s", pin.ID, pin.Label, pin.Type)

		switch pin.Label {
		case OnePPSLabel:
			OnePPSPin = pin
		case SMA1Label:
			SMA1Pin = pin
		}
	}

	log.Debugf("selectPin: found %d pins matching clockID %d", matchingPinCount, clockID)

	choosePPS := false
	if OnePPSPin != nil {
		choosePPS = true
		log.Debugf("selectPin: evaluating 1PPS pin ID=%d with %d parent devices", OnePPSPin.ID, len(OnePPSPin.ParentDevices))
		for _, parentDev := range OnePPSPin.ParentDevices {
			log.Debugf("selectPin: 1PPS parent device: direction=%s state=%s", parentDev.Direction, parentDev.State)
			if parentDev.State != ConnectedState {
				log.Debugf("selectPin: 1PPS rejected - parent device not connected (state=%s)", parentDev.State)
				choosePPS = false
				break
			}
		}
		if choosePPS {
			log.Debugf("selectPin: 1PPS pin selected")
		}
	} else {
		log.Debugf("selectPin: no 1PPS pin found")
	}

	chooseSMA1 := false
	if SMA1Pin != nil {
		chooseSMA1 = true
		log.Debugf("selectPin: evaluating SMA1 pin ID=%d with %d parent devices", SMA1Pin.ID, len(SMA1Pin.ParentDevices))
		for _, parentDev := range SMA1Pin.ParentDevices {
			log.Debugf("selectPin: SMA1 parent device: direction=%s state=%s", parentDev.Direction, parentDev.State)
			if parentDev.Direction != InputDirection || parentDev.State != ConnectedState {
				log.Debugf("selectPin: SMA1 rejected - parent device invalid (direction=%s, state=%s)", parentDev.Direction, parentDev.State)
				chooseSMA1 = false
				break
			}
		}
		if chooseSMA1 {
			log.Debugf("selectPin: SMA1 pin selected")
		}
	} else {
		log.Debugf("selectPin: no SMA1 pin found")
	}

	//nolint:gocritic // this is clearer
	if choosePPS {
		return OnePPSPin.ID, OnePPSLabel, nil
	}

	if chooseSMA1 {
		return SMA1Pin.ID, SMA1Label, nil
	}

	// Fallback: find any pin with at least one connected input parent device
	log.Debugf("selectPin: no 1PPS or SMA1 pin available, looking for any connected input pin")
	for _, pin := range entries {
		if pin.ClockID != clockID {
			continue
		}

		hasConnectedInput := false
		for _, parentDev := range pin.ParentDevices {
			if parentDev.Direction == InputDirection && parentDev.State == ConnectedState {
				hasConnectedInput = true
				break
			}
		}

		if hasConnectedInput {
			log.Infof("selectPin: using fallback pin ID=%d label=%s type=%s (no 1PPS/SMA1 available)", pin.ID, pin.Label, pin.Type)
			return pin.ID, pin.Label, nil
		}
	}

	return 0, "", utils.NewRequirementsNotMetError(errors.New("failed to determine correct offset pin: no suitable 1PPS or SMA1 pin found"))
}

func postProcessDPLLNetlinkClockID(result map[string]string) (map[string]any, error) {
	processedResult := make(map[string]any)

	clockID, err := strconv.ParseUint(result["dpll-netlink-clock-serial-number"], 16, 64)
	if err != nil {
		return processedResult, fmt.Errorf("failed to parse int for clock id: %w", err)
	}

	processedResult["clockID"] = clockID

	pinsJSON := result["dpll-netlink-pins"]
	if strings.Contains(pinsJSON, "has no attribute with value") {
		return processedResult, fmt.Errorf(
			"dpll tool does not support a netlink attribute reported by the firmware — "+
				"update the container image (set NETLINK_DEBUG_CONTAINER_IMAGE env var "+
				"to a newer version): %s",
			pinsJSON,
		)
	}

	// Try to select a pin using the NIC's clock ID first
	offsetPintID, pinType, err := selectPin([]byte(pinsJSON), clockID)

	// If no pins match the NIC's clock ID, try using the clock ID from DPLL devices
	// This handles cases where DPLL is on a separate timing card (e.g., zl3073x)
	if err != nil {
		var reqNotMet *utils.RequirementsNotMetError
		if errors.As(err, &reqNotMet) {
			log.Debugf("No pins found for NIC clockID %d, trying DPLL device clock IDs", clockID)

			devices, devErr := parseDeviceJSON([]byte(result["dpll-netlink-devices"]))
			if devErr == nil && len(devices) > 0 {
				fallbackClockID := devices[0].ClockID
				log.Infof("Using DPLL device clock ID %d (module: %s) instead of NIC clock ID %d",
					fallbackClockID, devices[0].Driver, clockID)

				offsetPintID, pinType, err = selectPin([]byte(pinsJSON), fallbackClockID)
				if err == nil {
					processedResult["clockID"] = fallbackClockID
				}
			}
		}
	}

	if err != nil {
		return processedResult, err
	}

	processedResult["offsetPin"] = offsetPintID
	processedResult["pinType"] = pinType

	return processedResult, nil
}

type NetlinkParameters struct {
	Timestamp string `fetcherKey:"date"      json:"timestamp"`
	PinType   string `fetcherKey:"pinType"   json:"pinType"`
	ClockID   uint64 `fetcherKey:"clockID"   json:"clockId"`
	OffsetPin int32  `fetcherKey:"offsetPin" json:"offsetPin"`
}

func GetNetlinkParameters(ctx clients.ExecContext, interfaceName string) (NetlinkParameters, error) {
	netlinkInfo := NetlinkParameters{}

	fetcherInst, fetchedInstanceOk := dpllClockIDFetcher[interfaceName]
	if !fetchedInstanceOk {
		err := BuildNetlinkInfoFetcher(interfaceName)
		if err != nil {
			return netlinkInfo, err
		}

		fetcherInst, fetchedInstanceOk = dpllClockIDFetcher[interfaceName]
		if !fetchedInstanceOk {
			return netlinkInfo, errors.New("failed to create fetcher for DPLLInfo using netlink interface")
		}
	}

	err := fetcherInst.Fetch(ctx, &netlinkInfo)
	if err != nil {
		return netlinkInfo, fmt.Errorf("failed to fetch netlink info %w", err)
	}

	return netlinkInfo, nil
}
