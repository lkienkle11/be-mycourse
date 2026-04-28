package helper

import "strings"

func ShouldEnqueueSupersededCloudCleanup(prevObjectKey, prevBunnyVideoID, newObjectKey, newBunnyVideoID string) bool {
	pk := strings.TrimSpace(prevObjectKey)
	nk := strings.TrimSpace(newObjectKey)
	pbv := strings.TrimSpace(prevBunnyVideoID)
	nbv := strings.TrimSpace(newBunnyVideoID)
	if pk == "" || nk == "" {
		return false
	}
	if pbv != "" || nbv != "" {
		return pbv != nbv || pk != nk
	}
	return pk != nk
}
