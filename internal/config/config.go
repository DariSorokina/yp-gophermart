package config

import (
	"flag"
	"os"
)

type FlagConfig struct {
	FlagRunAddr              string
	FlagDatabaseURI          string
	FlagAccrualSystemAddress string
	FlagLogLevel             string
}

func NewFlagConfig() *FlagConfig {
	return &FlagConfig{}
}

func ParseFlags() (flagConfig *FlagConfig) {
	flagConfig = NewFlagConfig()
	flag.StringVar(&flagConfig.FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&flagConfig.FlagRunAddr, "a", "localhost:44335", "service launch address and port")
	flag.StringVar(&flagConfig.FlagDatabaseURI, "d", "host=localhost user=app password=123qwe dbname=orders_database sslmode=disable", "database connection address")
	flag.StringVar(&flagConfig.FlagAccrualSystemAddress, "r", "http://localhost:8080/", "accrual system address")
	flag.Parse()

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagConfig.FlagLogLevel = envLogLevel
	}
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		flagConfig.FlagRunAddr = envRunAddr
	}
	if envDatabaseURL := os.Getenv("DATABASE_URI"); envDatabaseURL != "" {
		flagConfig.FlagDatabaseURI = envDatabaseURL
	}
	if envAccrualSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualSystemAddress != "" {
		flagConfig.FlagAccrualSystemAddress = envAccrualSystemAddress
	}
	return
}
