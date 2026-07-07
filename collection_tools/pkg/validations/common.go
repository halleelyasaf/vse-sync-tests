// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

const (
	TGMTestIDBase   = "https://github.com/openshift-kni/vse-sync-tests/tree/main/tests"
	TGMEnvModelPath = TGMTestIDBase + "/environment/model"
	TGMEnvVerPath   = TGMTestIDBase + "/environment/version"
	TGMSyncEnvPath  = TGMTestIDBase + "/sync/G.8272/environment/status"
)

const (
	clusterVersionOrdering int = iota
	ptpOperatorVersionOrdering
	gpsdVersionOrdering
	deviceDetailsOrdering
	deviceDriverVersionOrdering
	deviceFirmwareOrdering
	gnssModuleOrdering
	gnssVersionOrdering
	gnssProtOrdering
	hasGNSSDevicesOrdering
	gnssConnectedToAntOrdering
	gnssReceivingDataOrdering
	configuredForGrandMasterOrdering
)

type VersionCheck struct {
	id           string `json:"-"`
	Version      string `json:"version"`
	checkVersion string `json:"-"`
	MinVersion   string `json:"expected"`
	description  string `json:"-"`
	order        int    `json:"-"`
}

// normalizeVersion strips leading zeros from each dot-separated component
// so that firmware versions like "5.00" become "5.0" (valid semver).
func normalizeVersion(v string) string {
	parts := strings.Split(v, ".")
	for i, p := range parts {
		if n, err := strconv.Atoi(p); err == nil {
			parts[i] = strconv.Itoa(n)
		}
	}
	return strings.Join(parts, ".")
}

func (verCheck *VersionCheck) Verify() error {
	normalized := normalizeVersion(strings.ReplaceAll(verCheck.checkVersion, "_", "-"))
	ver := "v" + normalized
	if !semver.IsValid(ver) {
		return fmt.Errorf("could not parse version %s", ver)
	}

	minNormalized := normalizeVersion(verCheck.MinVersion)
	if semver.Compare(ver, "v"+minNormalized) < 0 {
		return utils.NewInvalidEnvError(
			fmt.Errorf("unexpected version: %s < %s", verCheck.checkVersion, verCheck.MinVersion),
		)
	}

	return nil
}

func (verCheck *VersionCheck) GetID() string {
	return verCheck.id
}

func (verCheck *VersionCheck) GetDescription() string {
	return verCheck.description
}

func (verCheck *VersionCheck) GetData() any { //nolint:ireturn // data will vary for each validation
	return verCheck
}

func (verCheck *VersionCheck) GetOrder() int {
	return verCheck.order
}

type VersionWithError struct {
	Error   error  `json:"fetchError"`
	Version string `json:"version"`
}

func MarshalVersionAndError(ver *VersionWithError) ([]byte, error) {
	var err any
	if ver.Error != nil {
		err = ver.Error.Error()
	}

	marsh, marshalErr := json.Marshal(&struct {
		Error   any    `json:"fetchError"`
		Version string `json:"version"`
	}{
		Version: ver.Version,
		Error:   err,
	})

	return marsh, fmt.Errorf("failed to marshal VersionWithError %w", marshalErr)
}

type VersionWithErrorCheck struct {
	Error error
	VersionCheck
}

func (verCheck *VersionWithErrorCheck) MarshalJSON() ([]byte, error) {
	return MarshalVersionAndError(&VersionWithError{
		Version: verCheck.Version,
		Error:   verCheck.Error,
	})
}

func (verCheck *VersionWithErrorCheck) Verify() error {
	if verCheck.Error != nil {
		return verCheck.Error
	}

	return verCheck.VersionCheck.Verify()
}
