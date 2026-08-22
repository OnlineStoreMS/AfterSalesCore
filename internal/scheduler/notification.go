package scheduler

import (
	"context"
	"log"
	"time"

	"aftersalescore/internal/service"
)

type NotificationScheduler struct {
	svc    *service.NotificationService
	stopCh chan struct{}
}

func NewNotificationScheduler(svc *service.NotificationService) *NotificationScheduler {
	return &NotificationScheduler{svc: svc, stopCh: make(chan struct{})}
}

func (s *NotificationScheduler) Start() {
	go s.loop()
}

func (s *NotificationScheduler) Stop() {
	close(s.stopCh)
}

func (s *NotificationScheduler) loop() {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			s.runOnce()
			timer.Reset(s.svc.MinPollInterval())
		}
	}
}

func (s *NotificationScheduler) runOnce() {
	if !s.svc.AnyEnabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	sent, skipped, err := s.svc.RunAll(ctx)
	if err != nil {
		log.Printf("[notification] poll failed: %v", err)
		return
	}
	if sent > 0 {
		log.Printf("[notification] sent %d message(s), skipped %d", sent, skipped)
	}
}
