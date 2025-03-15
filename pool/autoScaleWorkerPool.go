package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMinWorkers   = 1               // 預設最小 worker 數量
	DefaultMinQueueSize = 1               // 預設最小佇列大小
	DefaultIdleTimeout  = 5 * time.Second // worker 閒置多久後自動退出
)

type AutoScaleWorkerPool[T any] struct {
	Ctx            context.Context
	CancelFunc     context.CancelFunc
	MinWorkers     int32         // 最低保留 worker 數量
	MaxWorkers     int32         // 最大 worker 數量
	CurrentWorkers int32         // 目前 worker 數量
	QueueSize      int32         // 工作佇列大小
	ScaleThreshold int32         // 1-100 之間的數字，表示佇列長度超過多少比例時，應該動態增加 worker
	IdleTimeout    time.Duration // worker 閒置多久後自動退出
	InputChan      <-chan T
	WorkFunc       func(context.Context, T) // 工作函數
	WG             sync.WaitGroup
	Queue          chan T
}

func NewAutoScaleWorkerPool[T any](ctx context.Context, minWorkers, maxWorkers, queueSize, scaleThreshold int32, idleTimeout time.Duration, InputChan <-chan T, workFunc func(context.Context, T)) (pool *AutoScaleWorkerPool[T]) {
	pool = &AutoScaleWorkerPool[T]{
		MinWorkers:     max(DefaultMinWorkers, minWorkers),
		MaxWorkers:     max(DefaultMinWorkers, maxWorkers),
		QueueSize:      max(DefaultMinQueueSize, queueSize),
		ScaleThreshold: min(max(scaleThreshold, 1), 100),
		IdleTimeout:    max(DefaultIdleTimeout, idleTimeout),
		InputChan:      InputChan,
		WorkFunc:       workFunc,
		WG:             sync.WaitGroup{},
	}
	pool.CurrentWorkers = pool.MinWorkers
	pool.Ctx, pool.CancelFunc = context.WithCancel(ctx)
	pool.Queue = make(chan T, pool.QueueSize)

	for range pool.CurrentWorkers {
		pool.WG.Add(1)
		go pool.worker()
	}

	go pool.Start()
	return
}

func (pool *AutoScaleWorkerPool[T]) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				qLen := int32(len(pool.Queue))
				currentWorkers := atomic.LoadInt32(&pool.CurrentWorkers)

				thresholdCount := pool.QueueSize * pool.ScaleThreshold / 100
				if qLen >= thresholdCount && currentWorkers < pool.MaxWorkers {
					// 超出門檻的任務數量
					excess := qLen - thresholdCount

					// 最大可能的超出量，即隊列從門檻滿到完全滿
					maxExcess := pool.QueueSize - thresholdCount

					// 當隊列完全滿時，比例為 1；否則依比例計算
					var ratio float64
					if maxExcess == 0 {
						ratio = 1
					} else {
						ratio = float64(excess) / float64(maxExcess)
					}

					// 根據比例計算可增加的 worker 數量，
					// 乘上剩餘可增加的 worker 總數 (MaxWorkers - currentWorkers)
					potentialAdditional := int32(ratio * float64(pool.MaxWorkers-currentWorkers))

					// 保證至少新增 1 個 worker
					additional := max(1, potentialAdditional)

					// 避免超過最大 worker 數量
					if currentWorkers+additional > pool.MaxWorkers {
						additional = pool.MaxWorkers - currentWorkers
					}

					for range additional {
						pool.WG.Add(1)
						go pool.worker()

					}
					atomic.AddInt32(&pool.CurrentWorkers, additional)
				}
			case <-pool.Ctx.Done():
				return
			}
		}
	}()

	pool.readChan()

	close(pool.Queue)
	pool.WG.Wait()
}

func (pool *AutoScaleWorkerPool[T]) readChan() {
	for {
		select {
		case data, ok := <-pool.InputChan:
			if !ok {
				return
			}
			// 當有資料時，再透過 select 將資料寫入 queue，避免寫入時也阻塞
			select {
			case pool.Queue <- data:
			case <-pool.Ctx.Done():
				return
			}
		case <-pool.Ctx.Done():
			return
		}
	}
}

func (pool *AutoScaleWorkerPool[T]) Stop() {
	pool.CancelFunc()
}

func (pool *AutoScaleWorkerPool[T]) worker() {
	defer func() {
		atomic.AddInt32(&pool.CurrentWorkers, -1)
		pool.WG.Done()
	}()

	timer := time.NewTimer(pool.IdleTimeout)
	defer timer.Stop()

	for {
		select {
		case payload, ok := <-pool.Queue:
			if !ok {
				return
			}
			// 收到工作後先停止計時器（避免計時器通道阻塞）
			if !timer.Stop() {
				<-timer.C
			}
			// 重設 idle timeout
			timer.Reset(pool.IdleTimeout)
			pool.WorkFunc(pool.Ctx, payload)
		case <-timer.C:
			// 僅在 worker 數超過 MinWorkers 時退出
			if atomic.LoadInt32(&pool.CurrentWorkers) > pool.MinWorkers {
				return
			} else {
				// 若已達最小 worker 數，則重設計時器，保持等待狀態
				timer.Reset(pool.IdleTimeout)
			}
		case <-pool.Ctx.Done():
			return
		}
	}
}
