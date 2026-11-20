package scheduler

import (
	"context"
	"github.com/temlaka/coinwatch/internal/usecase/schedule"
	"log"
	"time"
)

const retryDelay = 5 * time.Second

func UpdateAllPrice(ctx context.Context, svc *schedule.Service) {
	for {

		settings, err := svc.Get()
		if err != nil {
			log.Printf("get schedule: %v (retry in %s)", err, retryDelay)
			select {
			case <-ctx.Done():
				log.Println("schedule stopped")
				return
			case <-time.After(retryDelay):
				continue
			}
		}

		if !settings.Enabled {
			select {
			case <-ctx.Done():
				log.Println("schedule stopped")
				return

			case <-svc.Changes():
				continue
			}
		}
		select {
		case <-ctx.Done():
			log.Println("schedule stopped")
			return

		case <-svc.Changes():
			continue
		case <-time.After(time.Until(settings.NextUpdate)):
			_, _, err := svc.Trigger()
			if err != nil {
				log.Printf("trigger: %v", err)
			}
		}
	}
}
