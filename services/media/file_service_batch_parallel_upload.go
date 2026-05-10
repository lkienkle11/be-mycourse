package media

import (
	"sync"

	"golang.org/x/sync/errgroup"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
)

// scheduleParallelUpload runs fn inside errgroup with bounded concurrency (shared semaphore).
func scheduleParallelUpload(g *errgroup.Group, sem chan struct{}, fn func() error) {
	g.Go(func() error {
		sem <- struct{}{}
		defer func() { <-sem }()
		return fn()
	})
}

func uploadPreparedCreatesParallel(clients *entities.CloudClients, prepared []entities.PreparedCreatePart) ([]entities.ProviderUploadResult, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	results := make([]entities.ProviderUploadResult, len(prepared))
	var mu sync.Mutex
	var finished []int

	g := new(errgroup.Group)
	sem := make(chan struct{}, constants.MaxConcurrentMediaUploadWorkers)
	for i := range prepared {
		i := i
		scheduleParallelUpload(g, sem, func() error {
			return runSinglePreparedUpload(clients, prepared, results, &mu, &finished, i)
		})
	}
	if err := g.Wait(); err != nil {
		mu.Lock()
		idxs := append([]int(nil), finished...)
		mu.Unlock()
		for _, idx := range idxs {
			deleteUploadAttempt(clients, prepared[idx].Provider, prepared[idx].ObjectKey, results[idx])
		}
		return nil, err
	}
	return results, nil
}

func runSinglePreparedUpload(
	clients *entities.CloudClients,
	prepared []entities.PreparedCreatePart,
	results []entities.ProviderUploadResult,
	mu *sync.Mutex,
	finished *[]int,
	i int,
) error {
	if MediaUploadParallelStartProbe != nil {
		MediaUploadParallelStartProbe()
	}
	r, err := uploadToProvider(clients, prepared[i].Provider, prepared[i].ObjectKey, prepared[i].Filename, prepared[i].Payload, entities.RawMetadata{})
	if err != nil {
		return err
	}
	mu.Lock()
	results[i] = r
	*finished = append(*finished, i)
	mu.Unlock()
	return nil
}
