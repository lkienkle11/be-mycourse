package utils

import "mycourse-io-be/constants"

// imageEncodeGate is a buffered-channel semaphore that caps concurrent WebP encode goroutines
// to constants.MaxConcurrentImageEncode per process.
//
// Usage pattern (always paired):
//
//	AcquireEncodeGate()
//	defer ReleaseEncodeGate()
//	encoded, mime, err := EncodeWebP(payload)
var imageEncodeGate = make(chan struct{}, constants.MaxConcurrentImageEncode)

// AcquireEncodeGate blocks until one gate slot is available.
// Every call MUST be paired with a deferred ReleaseEncodeGate.
func AcquireEncodeGate() { imageEncodeGate <- struct{}{} }

// ReleaseEncodeGate returns the previously acquired slot back to the gate.
func ReleaseEncodeGate() { <-imageEncodeGate }
