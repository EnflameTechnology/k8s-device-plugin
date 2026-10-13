// Copyright (c) 2021 - 2022, NVIDIA CORPORATION. All rights reserved.

package mig

import (
	"bufio"
	"fmt"
	"os"

	"k8s.io/klog/v2"
)

const (
	enflameProcDriverPath   = "/proc/driver/enflame"
	enflameCapabilitiesPath = enflameProcDriverPath + "/capabilities"

	nvcapsProcDriverPath = "/proc/driver/enflame-caps"
	nvcapsMigMinorsPath  = nvcapsProcDriverPath + "/mig-minors"
	nvcapsDevicePath     = "/dev/enflame-caps"
)

// GetMigCapabilityDevicePaths returns a mapping of MIG capability path to device node path
func GetMigCapabilityDevicePaths() (map[string]string, error) {
	// Open nvcapsMigMinorsPath for walking.
	// If the nvcapsMigMinorsPath does not exist, then we are not on a MIG
	// capable machine, so there is nothing to do.
	// The format of this file is discussed in:
	//     https://docs.enflame.com/datacenter/tesla/mig-user-guide/index.html#unique_1576522674
	minorsFile, err := os.Open(nvcapsMigMinorsPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error opening MIG minors file: %v", err)
	}
	defer minorsFile.Close()

	// Define a function to process each each line of nvcapsMigMinorsPath
	processLine := func(line string) (string, int, error) {
		var gcu, gi, ci, migMinor int

		// Look for a CI access file
		n, _ := fmt.Sscanf(line, "gcu%d/gi%d/ci%d/access %d", &gcu, &gi, &ci, &migMinor)
		if n == 4 {
			capPath := fmt.Sprintf(enflameCapabilitiesPath+"/gcu%d/mig/gi%d/ci%d/access", gcu, gi, ci)
			return capPath, migMinor, nil
		}

		// Look for a GI access file
		n, _ = fmt.Sscanf(line, "gcu%d/gi%d/access %d", &gcu, &gi, &migMinor)
		if n == 3 {
			capPath := fmt.Sprintf(enflameCapabilitiesPath+"/gcu%d/mig/gi%d/access", gcu, gi)
			return capPath, migMinor, nil
		}

		// Look for the MIG config file
		n, _ = fmt.Sscanf(line, "config %d", &migMinor)
		if n == 1 {
			capPath := fmt.Sprintf(enflameCapabilitiesPath + "/mig/config")
			return capPath, migMinor, nil
		}

		// Look for the MIG monitor file
		n, _ = fmt.Sscanf(line, "monitor %d", &migMinor)
		if n == 1 {
			capPath := fmt.Sprintf(enflameCapabilitiesPath + "/mig/monitor")
			return capPath, migMinor, nil
		}

		return "", 0, fmt.Errorf("unparsable line: %v", line)
	}

	// Walk each line of nvcapsMigMinorsPath and construct a mapping of enflame
	// capabilities path to device minor for that capability
	capsDevicePaths := make(map[string]string)
	scanner := bufio.NewScanner(minorsFile)
	for scanner.Scan() {
		capPath, migMinor, err := processLine(scanner.Text())
		if err != nil {
			klog.Errorf("Skipping line in MIG minors file: %v", err)
			continue
		}
		capsDevicePaths[capPath] = fmt.Sprintf(nvcapsDevicePath+"/enflame-cap%d", migMinor)
	}
	return capsDevicePaths, nil
}
