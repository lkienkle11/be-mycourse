package media

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	pkgmedia "mycourse-io-be/pkg/media"
)

func deleteUploadAttempt(clients *entities.CloudClients, provider, objectKey string, uploaded entities.ProviderUploadResult) {
	bunny := ""
	if uploaded.Metadata != nil {
		if v := uploaded.Metadata[constants.MediaMetaKeyBunnyVideoID]; v != nil {
			bunny = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if bunny == "" && uploaded.Metadata[constants.MediaMetaKeyVideoGUID] != nil {
			bunny = strings.TrimSpace(fmt.Sprintf("%v", uploaded.Metadata[constants.MediaMetaKeyVideoGUID]))
		}
	}
	_ = pkgmedia.DeleteStoredObject(context.Background(), clients, objectKey, provider, bunny)
}

func enqueueBundleTailJobs(
	scheduleUpload func(func() error),
	clients *entities.CloudClients,
	tail []entities.PreparedCreatePart,
	tailResults []entities.ProviderUploadResult,
	mu *sync.Mutex,
	tailFinished *[]int,
) {
	for i := range tail {
		i := i
		scheduleUpload(func() error {
			return runBundleTailUpload(clients, tail, tailResults, mu, tailFinished, i)
		})
	}
}

func uploadBundleParallel(clients *entities.CloudClients, head *entities.PreparedUpdateHead, tail []entities.PreparedCreatePart) (entities.ProviderUploadResult, []entities.ProviderUploadResult, error) {
	var headResult entities.ProviderUploadResult
	tailResults := make([]entities.ProviderUploadResult, len(tail))

	var mu sync.Mutex
	var headFinished bool
	var tailFinished []int

	g := new(errgroup.Group)
	sem := make(chan struct{}, constants.MaxConcurrentMediaUploadWorkers)

	scheduleUpload := func(fn func() error) {
		scheduleParallelUpload(g, sem, fn)
	}

	if head != nil {
		scheduleUpload(func() error {
			return runBundleHeadUpload(clients, head, &mu, &headFinished, &headResult)
		})
	}

	enqueueBundleTailJobs(scheduleUpload, clients, tail, tailResults, &mu, &tailFinished)

	if err := g.Wait(); err != nil {
		cleanupAfterBundleUploadFailure(clients, head, headFinished, headResult, tail, tailResults, tailFinished)
		return entities.ProviderUploadResult{}, nil, err
	}
	return headResult, tailResults, nil
}

func runBundleHeadUpload(
	clients *entities.CloudClients,
	head *entities.PreparedUpdateHead,
	mu *sync.Mutex,
	headFinished *bool,
	headResult *entities.ProviderUploadResult,
) error {
	if MediaUploadParallelStartProbe != nil {
		MediaUploadParallelStartProbe()
	}
	r, err := uploadToProvider(clients, head.Provider, head.ResolvedObjectKey, head.FilenameNorm, head.PayloadNorm, entities.RawMetadata{})
	if err != nil {
		return err
	}
	mu.Lock()
	*headResult = r
	*headFinished = true
	mu.Unlock()
	return nil
}

func runBundleTailUpload(
	clients *entities.CloudClients,
	tail []entities.PreparedCreatePart,
	tailResults []entities.ProviderUploadResult,
	mu *sync.Mutex,
	tailFinished *[]int,
	i int,
) error {
	if MediaUploadParallelStartProbe != nil {
		MediaUploadParallelStartProbe()
	}
	r, err := uploadToProvider(clients, tail[i].Provider, tail[i].ObjectKey, tail[i].Filename, tail[i].Payload, entities.RawMetadata{})
	if err != nil {
		return err
	}
	mu.Lock()
	tailResults[i] = r
	*tailFinished = append(*tailFinished, i)
	mu.Unlock()
	return nil
}

func cleanupAfterBundleUploadFailure(
	clients *entities.CloudClients,
	head *entities.PreparedUpdateHead,
	headFinished bool,
	headResult entities.ProviderUploadResult,
	tail []entities.PreparedCreatePart,
	tailResults []entities.ProviderUploadResult,
	tailFinished []int,
) {
	if headFinished && head != nil {
		deleteUploadAttempt(clients, head.Provider, head.ResolvedObjectKey, headResult)
	}
	for _, idx := range tailFinished {
		deleteUploadAttempt(clients, tail[idx].Provider, tail[idx].ObjectKey, tailResults[idx])
	}
}
