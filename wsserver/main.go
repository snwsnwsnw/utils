package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"wsserver/server"
)

func main() {
	server.LoadWSAddrs()
	server.Start(context.Background())
	server.RedisSubscribe()

	//newPool := pool.NewAutoScaleWorkerPool(context.Background(), 10, 100, 1000, 30, time.Second*10)
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("收到終止訊號，關閉連線...")
}
