/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package tops

import (
	"unsafe"
)

/*
#cgo linux LDFLAGS: -Wl,--export-dynamic -Wl,--unresolved-symbols=ignore-in-object-files
#cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup

#ifdef _WIN32
#define TOPSAPI __stdcall
#else
#define TOPSAPI
#endif

typedef int TOPSdevice;

typedef enum TOPSdevice_attribute_enum {
    CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MAJOR = 75,
    CU_DEVICE_ATTRIBUTE_COMPUTE_CAPABILITY_MINOR = 76
} TOPSdevice_attribute;

typedef enum topsError_enum {
    TOPS_SUCCESS                              = 0,
    TOPS_ERROR_INVALID_VALUE                  = 1,
    TOPS_ERROR_OUT_OF_MEMORY                  = 2,
    TOPS_ERROR_NOT_INITIALIZED                = 3,
    TOPS_ERROR_DEINITIALIZED                  = 4,
    TOPS_ERROR_PROFILER_DISABLED              = 5,
    TOPS_ERROR_PROFILER_NOT_INITIALIZED       = 6,
    TOPS_ERROR_PROFILER_ALREADY_STARTED       = 7,
    TOPS_ERROR_PROFILER_ALREADY_STOPPED       = 8,
    TOPS_ERROR_NO_DEVICE                      = 100,
    TOPS_ERROR_INVALID_DEVICE                 = 101,
    TOPS_ERROR_INVALID_IMAGE                  = 200,
    TOPS_ERROR_INVALID_CONTEXT                = 201,
    TOPS_ERROR_CONTEXT_ALREADY_CURRENT        = 202,
    TOPS_ERROR_MAP_FAILED                     = 205,
    TOPS_ERROR_UNMAP_FAILED                   = 206,
    TOPS_ERROR_ARRAY_IS_MAPPED                = 207,
    TOPS_ERROR_ALREADY_MAPPED                 = 208,
    TOPS_ERROR_NO_BINARY_FOR_GCU              = 209,
    TOPS_ERROR_ALREADY_ACQUIRED               = 210,
    TOPS_ERROR_NOT_MAPPED                     = 211,
    TOPS_ERROR_NOT_MAPPED_AS_ARRAY            = 212,
    TOPS_ERROR_NOT_MAPPED_AS_POINTER          = 213,
    TOPS_ERROR_ECC_UNCORRECTABLE              = 214,
    TOPS_ERROR_UNSUPPORTED_LIMIT              = 215,
    TOPS_ERROR_CONTEXT_ALREADY_IN_USE         = 216,
    TOPS_ERROR_PEER_ACCESS_UNSUPPORTED        = 217,
    TOPS_ERROR_INVALID_PTX                    = 218,
    TOPS_ERROR_INVALID_GRAPHICS_CONTEXT       = 219,
    TOPS_ERROR_NVLINK_UNCORRECTABLE           = 220,
    TOPS_ERROR_JIT_COMPILER_NOT_FOUND         = 221,
    TOPS_ERROR_INVALID_SOURCE                 = 300,
    TOPS_ERROR_FILE_NOT_FOUND                 = 301,
    TOPS_ERROR_SHARED_OBJECT_SYMBOL_NOT_FOUND = 302,
    TOPS_ERROR_SHARED_OBJECT_INIT_FAILED      = 303,
    TOPS_ERROR_OPERATING_SYSTEM               = 304,
    TOPS_ERROR_INVALID_HANDLE                 = 400,
    TOPS_ERROR_NOT_FOUND                      = 500,
    TOPS_ERROR_NOT_READY                      = 600,
    TOPS_ERROR_ILLEGAL_ADDRESS                = 700,
    TOPS_ERROR_LAUNCH_OUT_OF_RESOURCES        = 701,
    TOPS_ERROR_LAUNCH_TIMEOUT                 = 702,
    TOPS_ERROR_LAUNCH_INCOMPATIBLE_TEXTURING  = 703,
    TOPS_ERROR_PEER_ACCESS_ALREADY_ENABLED    = 704,
    TOPS_ERROR_PEER_ACCESS_NOT_ENABLED        = 705,
    TOPS_ERROR_PRIMARY_CONTEXT_ACTIVE         = 708,
    TOPS_ERROR_CONTEXT_IS_DESTROYED           = 709,
    TOPS_ERROR_ASSERT                         = 710,
    TOPS_ERROR_TOO_MANY_PEERS                 = 711,
    TOPS_ERROR_HOST_MEMORY_ALREADY_REGISTERED = 712,
    TOPS_ERROR_HOST_MEMORY_NOT_REGISTERED     = 713,
    TOPS_ERROR_HARDWARE_STACK_ERROR           = 714,
    TOPS_ERROR_ILLEGAL_INSTRUCTION            = 715,
    TOPS_ERROR_MISALIGNED_ADDRESS             = 716,
    TOPS_ERROR_INVALID_ADDRESS_SPACE          = 717,
    TOPS_ERROR_INVALID_PC                     = 718,
    TOPS_ERROR_LAUNCH_FAILED                  = 719,
    TOPS_ERROR_COOPERATIVE_LAUNCH_TOO_LARGE   = 720,
    TOPS_ERROR_NOT_PERMITTED                  = 800,
    TOPS_ERROR_NOT_SUPPORTED                  = 801,
    TOPS_ERROR_UNKNOWN                        = 999
} CUresult;

CUresult TOPSAPI topsInit(unsigned int Flags);
CUresult TOPSAPI topsDriverGetVersion(int *driverVersion);
CUresult TOPSAPI topsDeviceGet(TOPSdevice *device, int ordinal);
CUresult TOPSAPI topsDeviceGetAttribute(int *pi, TOPSdevice_attribute attrib, TOPSdevice dev);
CUresult TOPSAPI topsDeviceGetCount(int *count);
CUresult TOPSAPI topsDeviceTotalMem(size_t *bytes, TOPSdevice dev);
CUresult TOPSAPI topsDeviceGetName(char *name, int len, TOPSdevice dev);
*/
import "C"

// topsInit function as declared in tops.h
func topsInit(flags uint32) Result {
	cFlags := (C.uint)(flags)
	_ret := C.topsInit(cFlags)

	return Result(_ret)
}

// topsDeviceGet function as declared in tops.h
func topsDeviceGet(device *Device, index int32) Result {
	cDevice := (*C.TOPSdevice)(unsafe.Pointer(device))
	cIndex := (C.int)(index)

	_ret := C.topsDeviceGet(cDevice, cIndex)

	return Result(_ret)
}

// topsDeviceGetAttribute function as declared in tops.h
func topsDeviceGetAttribute(value *int32, attribute DeviceAttribute, dev Device) Result {
	cValue := (*C.int)(unsafe.Pointer(value))
	cAttribute := (C.TOPSdevice_attribute)(attribute)
	cDev := (C.TOPSdevice)(dev)

	_ret := C.topsDeviceGetAttribute(cValue, cAttribute, cDev)

	return Result(_ret)
}

// topsDeviceGetCount function as declared in tops.h
func topsDeviceGetCount(count *int32) Result {
	cCount := (*C.int)(unsafe.Pointer(count))
	_ret := C.topsDeviceGetCount(cCount)

	return Result(_ret)
}

// topsDriverGetVersion function as declared in tops.h
func topsDriverGetVersion(version *int32) Result {
	cVersion := (*C.int)(version)
	_ret := C.topsDriverGetVersion(cVersion)

	return Result(_ret)
}

// topsDeviceTotalMem function as declared in tops.h
func topsDeviceTotalMem(bytes *uint64, dev Device) Result {
	cBytes := (*C.size_t)(unsafe.Pointer(bytes))
	cDev := (C.TOPSdevice)(dev)
	_ret := C.topsDeviceTotalMem(cBytes, cDev)

	return Result(_ret)
}

// topsDeviceGetName function as declared in tops.h
func topsDeviceGetName(name *byte, len int32, dev Device) Result {
	cName := (*C.char)(unsafe.Pointer(name))
	cLen := (C.int)(len)
	cDev := (C.TOPSdevice)(dev)
	_ret := C.topsDeviceGetName(cName, cLen, cDev)

	return Result(_ret)
}
