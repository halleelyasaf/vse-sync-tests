// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"errors"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

const (
	deviceDetailsID          = TGMEnvModelPath + "/nic/"
	deviceDetailsDescription = "Verify NIC device model"
)

// Intel CarterFlat E830 12d3
var (
	VendorIntel        = "0x8086"
	E810WesportChannel = "0x1593"
	E810LoganBeach     = "0x1592"
	E825CarterFlat     = "0x579e"
	E830CarterFlat     = "0x12d3"
)

type DeviceDetails struct {
	VendorID string `json:"vendorId"`
	DeviceID string `json:"deviceId"`
}

func (dev *DeviceDetails) Verify() error {
	if dev.VendorID != VendorIntel ||
		(dev.DeviceID != E810WesportChannel &&
			dev.DeviceID != E810LoganBeach &&
			dev.DeviceID != E825CarterFlat &&
			dev.DeviceID != E830CarterFlat) {
		return utils.NewInvalidEnvError(errors.New("NIC device is not based on E810/E825/E830"))
	}

	return nil
}

func (dev *DeviceDetails) GetID() string {
	return deviceDetailsID
}

func (dev *DeviceDetails) GetDescription() string {
	return deviceDetailsDescription
}

func (dev *DeviceDetails) GetData() any { //nolint:ireturn // data will very for each validation
	return dev
}

func (dev *DeviceDetails) GetOrder() int {
	return deviceDetailsOrdering
}

func NewDeviceDetails(ptpDevInfo *devices.PTPDeviceInfo) *DeviceDetails {
	return &DeviceDetails{
		VendorID: ptpDevInfo.VendorID,
		DeviceID: ptpDevInfo.DeviceID,
	}
}
