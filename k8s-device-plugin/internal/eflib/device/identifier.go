/*
 * Copyright (c) NVIDIA CORPORATION.  All rights reserved.
 * Copyright (c) 2024, ENFLAME INC.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package device

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Identifier can be used to refer to a GCU or MIG device.
// This includes a device index or UUID.
type Identifier string

// IsGcuIndex checks if an identifier is a full GCU index
func (i Identifier) IsGcuIndex() bool {
	if _, err := strconv.ParseUint(string(i), 10, 0); err != nil {
		return false
	}
	return true
}

// IsMigIndex checks if an identifier is a MIG index
func (i Identifier) IsMigIndex() bool {
	split := strings.Split(string(i), ":")
	if len(split) != 2 {
		return false
	}
	for _, s := range split {
		if !Identifier(s).IsGcuIndex() {
			return false
		}
	}
	return true
}

// IsUUID checks if an identifier is a UUID
func (i Identifier) IsUUID() bool {
	return i.IsGcuUUID() || i.IsMigUUID()
}

// IsGcuUUID checks if an identifier is a GCU UUID
// A GCU UUID must be of the form GCU-b1028956-cfa2-0990-bf4a-5da9abb51763
func (i Identifier) IsGcuUUID() bool {
	if !strings.HasPrefix(string(i), "GCU-") {
		return false
	}
	_, err := uuid.Parse(strings.TrimPrefix(string(i), "GCU-"))
	return err == nil
}

// IsMigUUID checks if an identifier is a MIG UUID
// A MIG UUID can be of one of two forms:
//   - MIG-b1028956-cfa2-0990-bf4a-5da9abb51763
//   - MIG-GCU-b1028956-cfa2-0990-bf4a-5da9abb51763/3/0
func (i Identifier) IsMigUUID() bool {
	if !strings.HasPrefix(string(i), "MIG-") {
		return false
	}
	suffix := strings.TrimPrefix(string(i), "MIG-")
	_, err := uuid.Parse(suffix)
	if err == nil {
		return true
	}
	split := strings.Split(suffix, "/")
	if len(split) != 3 {
		return false
	}
	if !Identifier(split[0]).IsGcuUUID() {
		return false
	}
	for _, s := range split[1:] {
		_, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return false
		}
	}
	return true
}
