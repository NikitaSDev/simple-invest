package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"simple-invest/internal/app"
	"simple-invest/internal/config"
	"simple-invest/internal/servicelog"
	"syscall"
	"time"
)

const (
	timeout = 20
)

func main() {
	defer servicelog.InfoLog().Print("Сервер остановлен")

	servicelog.InfoLog().Print("Подключение к базе данных установлено")

	cfg := config.MustLoad()
	app := app.New(cfg.StoragePath, cfg.Port)
	go func() {
		err := app.Run()
		if err != nil {
			log.Println(err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		log.Println("server aborted")
	}
}
