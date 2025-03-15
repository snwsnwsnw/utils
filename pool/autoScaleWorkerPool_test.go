package pool

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestBasicProcessing 測試基本的任務處理情境
func TestBasicProcessing(t *testing.T) {
	input := make(chan int, 20)
	var processed int32 = 0
	workFunc := func(ctx context.Context, task int) {
		atomic.AddInt32(&processed, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定 minWorkers=2, maxWorkers=5, queueSize=10, scaleThreshold=50%
	pool := NewAutoScaleWorkerPool(ctx, 2, 5, 10, 50, 5*time.Second, input, workFunc)

	// 傳入 10 個任務
	for i := 0; i < 10; i++ {
		input <- i
	}
	close(input)

	// 等待 pool 處理完所有任務
	pool.WG.Wait()
	if atomic.LoadInt32(&processed) != 10 {
		t.Errorf("預期處理 10 個任務，但實際處理 %d 個", processed)
	}
}

// TestScalingUp 測試自動擴充 worker 的情境
func TestScalingUp(t *testing.T) {
	input := make(chan int, 100)
	var processed int32 = 0
	// 利用 blockCh 阻塞 worker，讓任務堆積在佇列中
	blockCh := make(chan struct{})
	workFunc := func(ctx context.Context, task int) {
		<-blockCh // 阻塞直到測試釋放
		atomic.AddInt32(&processed, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定 minWorkers=1, maxWorkers=5, queueSize=10, scaleThreshold=50%
	pool := NewAutoScaleWorkerPool(ctx, 1, 5, 10, 50, 5*time.Second, input, workFunc)

	// 傳入 11 個任務
	for i := 0; i <= 10; i++ {
		input <- i
	}
	// 不關閉 input，等待 autoscale 機制運作（ticker 間隔固定 2 秒）
	time.Sleep(3 * time.Second)
	currentWorkers := atomic.LoadInt32(&pool.CurrentWorkers)
	// 預期已經擴充至 5 個 worker
	if currentWorkers != 5 {
		t.Errorf("預期自動擴充後 currentWorkers 為 5，但實際為 %d", currentWorkers)
	}

	// 釋放 worker 處理任務，並關閉 channel 以完成 pool 的結束
	close(blockCh)
	close(input)
	pool.WG.Wait()
	if atomic.LoadInt32(&processed) != 11 {
		t.Errorf("預期處理 11 個任務，但實際處理 %d 個", processed)
	}
}

// TestStop 測試 Stop 方法是否能正確停止 pool
func TestStop(t *testing.T) {
	input := make(chan int, 10)
	var processed int32 = 0
	workFunc := func(ctx context.Context, task int) {
		atomic.AddInt32(&processed, 1)
		time.Sleep(100 * time.Millisecond) // 模擬處理時間
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := NewAutoScaleWorkerPool(ctx, 2, 5, 10, 50, 5*time.Second, input, workFunc)

	// 傳入 5 個任務
	for i := 0; i < 5; i++ {
		input <- i
	}
	// 呼叫 Stop 取消 pool
	pool.Stop()
	close(input)
	pool.WG.Wait()

	// 停止後 processed 任務數可能少於或等於 5
	if atomic.LoadInt32(&processed) > 5 {
		t.Errorf("預期處理的任務數不超過 5，但實際處理 %d 個", processed)
	} else {
		t.Logf("Stop 後 processed 任務數為 %d", processed)
	}
}

// TestScaleThreshold100 測試當 scaleThreshold 設為 100 時不會發生除以 0 的問題
func TestScaleThreshold100(t *testing.T) {
	input := make(chan int, 10)
	var processed int32 = 0
	// 利用 blockCh 阻塞 worker
	blockCh := make(chan struct{})
	workFunc := func(ctx context.Context, task int) {
		<-blockCh
		atomic.AddInt32(&processed, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定 scaleThreshold 為 100，此時 thresholdCount = queueSize，
	// 根據修正邏輯應直接設定比例為 1，並擴充 worker 到最大數量 5
	pool := NewAutoScaleWorkerPool(ctx, 1, 5, 10, 100, 5*time.Second, input, workFunc)

	// 傳入 11 個任務
	for i := 0; i <= 10; i++ {
		input <- i
	}
	// 等待 autoscale 機制觸發
	time.Sleep(3 * time.Second)
	currentWorkers := atomic.LoadInt32(&pool.CurrentWorkers)
	if currentWorkers != 5 {
		t.Errorf("當 scaleThreshold 為 100 時，預期 currentWorkers 為 5，但實際為 %d", currentWorkers)
	}

	close(blockCh)
	close(input)
	pool.WG.Wait()
	if atomic.LoadInt32(&processed) != 11 {
		t.Errorf("預期處理 11 個任務，但實際處理 %d 個", processed)
	}
}

// TestScalingDown 測試在空閒時是否能正確縮減 worker 數量，不低於 MinWorkers
func TestScalingDown(t *testing.T) {
	input := make(chan int, 100)
	var processed int32 = 0
	// 利用 blockCh 阻塞 workFunc，使 autoscale 產生額外 worker
	blockCh := make(chan struct{})
	workFunc := func(ctx context.Context, task int) {
		<-blockCh // 阻塞，模擬長時間處理
		atomic.AddInt32(&processed, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定 minWorkers=2, maxWorkers=5, queueSize=10, scaleThreshold=50%，idleTimeout=500ms（加速測試）
	pool := NewAutoScaleWorkerPool(ctx, 2, 5, 10, 50, 5*time.Second, input, workFunc)

	// 傳入足夠任務，促使 autoscale 擴充至最大 worker 數
	for i := 0; i < 20; i++ {
		input <- i
	}
	// 此時不關閉 input，讓 workFunc 因 blockCh 阻塞，使得任務堆積並觸發 autoscale
	time.Sleep(3 * time.Second)
	currentWorkers := atomic.LoadInt32(&pool.CurrentWorkers)
	if currentWorkers != 5 {
		t.Errorf("預期自動擴充後 currentWorkers 為 5，但實際為 %d", currentWorkers)
	}

	// 釋放阻塞，讓所有任務可以完成
	close(blockCh)
	// 等待所有任務處理完畢（busy wait 最多 5 秒）
	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt32(&processed) < 20 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	// 額外等待一段時間，讓 worker 因 idle 而縮減（idleTimeout 為 5s，等待 7 秒足夠）
	time.Sleep(7 * time.Second)

	// 此時應縮減至 MinWorkers（即 2）
	currentWorkers = atomic.LoadInt32(&pool.CurrentWorkers)
	if currentWorkers != 2 {
		t.Errorf("預期空閒後 currentWorkers 縮減為 MinWorkers 2，但實際為 %d", currentWorkers)
	}

	// 清理測試：關閉輸入並等待所有 goroutine 結束
	close(input)
	pool.WG.Wait()
}
