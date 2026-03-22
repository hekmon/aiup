//go:build linux

package nvvf

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef void* (*nvapi_query_t)(uint32_t);

static void* load_nvapi() {
	return dlopen("libnvidia-api.so.1", RTLD_LAZY);
}

static void* get_nvapi_func(void* handle, const char* name) {
	return dlsym(handle, name);
}

static void* call_nvapi_query(void* func, uint32_t id) {
	return ((nvapi_query_t)func)(id);
}

static int32_t call_nvapi_func_0(void* func) {
	return ((int32_t(*)(void))func)();
}

static int32_t call_nvapi_func_2(void* func, void* arg1, void* arg2) {
	return ((int32_t(*)(void*, void*))func)(arg1, arg2);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// nvapiLinux holds the loaded library and query function.
type nvapiLinux struct {
	handle    unsafe.Pointer
	queryFunc unsafe.Pointer
}

// loadNvAPILinux loads the NvAPI library on Linux.
func loadNvAPILinux() (*nvapiLinux, error) {
	handle := C.load_nvapi()
	if handle == nil {
		return nil, fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required)")
	}

	queryFunc := C.get_nvapi_func(handle, C.CString("nvapi_QueryInterface"))
	if queryFunc == nil {
		C.dlclose(handle)
		return nil, fmt.Errorf("nvapi_QueryInterface not exported")
	}

	return &nvapiLinux{
		handle:    handle,
		queryFunc: queryFunc,
	}, nil
}

// close cleans up the loaded library.
func (n *nvapiLinux) close() {
	if n.handle != nil {
		C.dlclose(n.handle)
	}
}

// resolve returns the function pointer for the given NvAPI function ID.
func (n *nvapiLinux) resolve(id uint32) unsafe.Pointer {
	return C.call_nvapi_query(n.queryFunc, C.uint32_t(id))
}

// call0 calls an NvAPI function with 0 arguments.
// Returns int32 cast to uint32 (Linux uses negative error codes).
func (n *nvapiLinux) call0(fn unsafe.Pointer) uint32 {
	return uint32(int32(C.call_nvapi_func_0(fn)))
}

// call2 calls an NvAPI function with 2 arguments.
func (n *nvapiLinux) call2(fn unsafe.Pointer, arg1, arg2 unsafe.Pointer) uint32 {
	return uint32(int32(C.call_nvapi_func_2(fn, arg1, arg2)))
}
