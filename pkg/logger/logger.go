package logger

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

var Logger *log.Logger

func Init() {
	date := time.Now().Format("2006-01-02")

	if err := os.MkdirAll("logs", 0755); err != nil {
		panic(err)
	}

	path := filepath.Join("logs", "app-"+date+".log")

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)

	if err != nil {
		panic(err)
	}

	Logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
}

func Info(v ...any) {
	Logger.Println(v...)
}

func Error(v ...any) {
	Logger.Println(v...)
}
