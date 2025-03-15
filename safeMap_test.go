package sutils

import (
	"sync"
	"testing"
)

func TestSafeMapWithSizeBasic(t *testing.T) {
	// 使用初始容量建立 SafeMap
	sm := NewSafeMap[int, string](100)
	if l := sm.Len(); l != 0 {
		t.Fatalf("預期初始化長度為 0，實際為 %d", l)
	}

	// 測試 Set 與 Get
	sm.Set(1, "one")
	value, ok := sm.Get(1)
	if !ok {
		t.Fatalf("預期可以取得 key 1 的值")
	}
	if value != "one" {
		t.Fatalf("預期 value 為 'one', 實際為 %v", value)
	}

	// 測試 Delete
	sm.Delete(1)
	_, ok = sm.Get(1)
	if ok {
		t.Fatalf("預期 key 1 已被刪除")
	}
}

func TestSafeMapWithoutSizeBasic(t *testing.T) {
	// 不傳入初始容量的情況
	sm := NewSafeMap[int, string]()
	if l := sm.Len(); l != 0 {
		t.Fatalf("預期初始化長度為 0，實際為 %d", l)
	}

	// 測試 Set 與 Get
	sm.Set(2, "two")
	value, ok := sm.Get(2)
	if !ok {
		t.Fatalf("預期可以取得 key 2 的值")
	}
	if value != "two" {
		t.Fatalf("預期 value 為 'two', 實際為 %v", value)
	}

	// 測試 Delete
	sm.Delete(2)
	_, ok = sm.Get(2)
	if ok {
		t.Fatalf("預期 key 2 已被刪除")
	}
}

func TestSafeMapConcurrency(t *testing.T) {
	// 使用初始容量建立 SafeMap
	sm := NewSafeMap[int, int](1000)
	var wg sync.WaitGroup

	// 並發寫入
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sm.Set(i, i)
		}(i)
	}
	wg.Wait()

	// 檢查長度是否正確
	if l := sm.Len(); l != 1000 {
		t.Fatalf("預期長度為 1000，實際為 %d", l)
	}

	// 並發讀取
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			val, ok := sm.Get(i)
			if !ok {
				t.Errorf("預期能取得 key %d", i)
			}
			if val != i {
				t.Errorf("預期 key %d 的值為 %d，實際為 %d", i, i, val)
			}
		}(i)
	}
	wg.Wait()

	// 並發刪除
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sm.Delete(i)
		}(i)
	}
	wg.Wait()

	// 最後檢查 SafeMap 是否為空
	if l := sm.Len(); l != 0 {
		t.Fatalf("預期長度為 0，實際為 %d", l)
	}
}

func TestSafeMapBasic(t *testing.T) {
	// 建立一個 SafeMap[int, string]
	sm := NewSafeMap[int, string]()

	// 測試 Set 與 Get
	sm.Set(1, "one")
	value, ok := sm.Get(1)
	if !ok {
		t.Fatalf("預期可以取得 key 1 的值")
	}
	if value != "one" {
		t.Fatalf("預期 value 為 'one', 實際為 %v", value)
	}

	// 測試 Delete
	sm.Delete(1)
	_, ok = sm.Get(1)
	if ok {
		t.Fatalf("預期 key 1 已被刪除")
	}
}

func TestSafeMapRange(t *testing.T) {
	// 建立一個 SafeMap 並插入一些測試資料
	sm := NewSafeMap[int, int](10)
	for i := 0; i < 10; i++ {
		sm.Set(i, i*i) // value 為 i 的平方
	}

	// 使用 Range 方法累加所有的 value
	sum := 0
	sm.Range(func(key int, value int) bool {
		sum += value
		return true // 繼續遍歷所有元素
	})

	// 預期值為 0^2 + 1^2 + 2^2 + ... + 9^2
	expected := 0
	for i := 0; i < 10; i++ {
		expected += i * i
	}

	if sum != expected {
		t.Fatalf("Range 遍歷結果錯誤，預期 %d，實際 %d", expected, sum)
	}
}

func TestSafeMapDeleteMultiple(t *testing.T) {
	// 建立一個 SafeMap 並設定多個鍵值對
	sm := NewSafeMap[int, string]()
	keys := []int{1, 2, 3, 4, 5}
	for _, key := range keys {
		sm.Set(key, "value")
	}

	// 刪除多個 key
	sm.Delete(2, 4, 5)

	// 驗證被刪除的 key 已不存在，其他仍存在
	if _, ok := sm.Get(2); ok {
		t.Errorf("預期 key 2 已被刪除")
	}
	if _, ok := sm.Get(4); ok {
		t.Errorf("預期 key 4 已被刪除")
	}
	if _, ok := sm.Get(5); ok {
		t.Errorf("預期 key 5 已被刪除")
	}

	// 驗證未刪除的 key 還是存在
	if val, ok := sm.Get(1); !ok || val != "value" {
		t.Errorf("預期 key 1 存在且 value 為 'value'")
	}
	if val, ok := sm.Get(3); !ok || val != "value" {
		t.Errorf("預期 key 3 存在且 value 為 'value'")
	}

	// 最後檢查 SafeMap 的長度應為剩下的 2 筆
	if l := sm.Len(); l != 2 {
		t.Fatalf("預期長度為 2，實際為 %d", l)
	}
}

// TestUnsafeMap 直接使用內建 map 並發寫入與讀取，預期在 race detector 下會出現競爭狀況，這個 test 一定會出現 FAIL 狀態才是正常的。
//func TestUnsafeMap(t *testing.T) {
//	// 建立一個普通的 map，不加任何同步機制
//	m := make(map[int]int)
//	var wg sync.WaitGroup
//
//	// 並發寫入
//	for i := 0; i < 1000; i++ {
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			m[i] = i
//		}(i)
//	}
//	wg.Wait()
//
//	// 並發讀取
//	for i := 0; i < 1000; i++ {
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			_ = m[i]
//		}(i)
//	}
//	wg.Wait()
//}
