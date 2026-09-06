package scheduler

import (
	"log"
	"time"

	"aftersalescore/internal/service"
)

// AgentSyncScheduler 由售后中心按采集间隔向 Agents 中心下发执行单（含上报地址等参数）。
type AgentSyncScheduler struct {
	svc    *service.ShopService
	stopCh chan struct{}
}

func NewAgentSyncScheduler(svc *service.ShopService) *AgentSyncScheduler {
	return &AgentSyncScheduler{svc: svc, stopCh: make(chan struct{})}
}

func (s *AgentSyncScheduler) Start() {
	go s.loop()
}

func (s *AgentSyncScheduler) Stop() {
	close(s.stopCh)
}

func (s *AgentSyncScheduler) loop() {
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			n, err := s.svc.DispatchDueAgentJobs()
			if err != nil {
				log.Printf("[agent-sync] dispatch failed: %v", err)
			} else if n > 0 {
				log.Printf("[agent-sync] dispatched %d job(s)", n)
			}
			timer.Reset(2 * time.Minute)
		}
	}
}
