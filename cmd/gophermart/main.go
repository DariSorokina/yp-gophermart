package main

import (
	"log"

	"github.com/DariSorokina/yp-gophermart.git/internal/app"
	"github.com/DariSorokina/yp-gophermart.git/internal/client"
	"github.com/DariSorokina/yp-gophermart.git/internal/config"
	"github.com/DariSorokina/yp-gophermart.git/internal/database"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/DariSorokina/yp-gophermart.git/internal/server"
)

func main() {
	flagConfig := config.ParseFlags()

	var l *logger.Logger
	var err error
	if l, err = logger.CreateLogger(flagConfig.FlagLogLevel); err != nil {
		log.Fatal("Failed to create logger:", err)
	}

	storage, err := database.NewPostgresqlDB(flagConfig.FlagDatabaseURI, l)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	app := app.NewApp(storage, l)
	accuralSystem := client.NewAccrualSystem(flagConfig.FlagAccrualSystemAddress, app, l)
	go client.Run(accuralSystem)

	serv := server.NewServer(app, flagConfig, l)
	if err := server.Run(serv); err != nil {
		panic(err)
	}

}
