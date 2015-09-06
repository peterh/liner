package liner

var procHeapAlloc = kernel32.NewProc("HeapAlloc")
var procHeapFree = kernel32.NewProc("HeapFree")
var procGetProcessHeap = kernel32.NewProc("GetProcessHeap")
var processHeap, _, _ = procGetProcessHeap.Call()

const heapZeroMemory = 0x8

func malloc(size uintptr) uintptr {
	ptr, _, _ := procHeapAlloc.Call(processHeap, heapZeroMemory, size)
	return ptr
}

func free(ptr uintptr) {
	if ptr != 0 {
		procHeapFree.Call(processHeap, 0, ptr)
	}
}
