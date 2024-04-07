package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/DariSorokina/yp-gophermart.git/internal/app"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/DariSorokina/yp-gophermart.git/internal/models"
)

var (
	ErrNoContent                 = errors.New("accural system: no content")
	ErrAccSysTooManyRequests     = errors.New("accrual system: too many requests")
	ErrAccSysInternalServerError = errors.New("accrual system: internal server error")
)

type AccrualSystem struct {
	accrualSystemAddress string
	app                  *app.App
	log                  *logger.Logger
}

func NewAccrualSystem(accrualSystemAddress string, app *app.App, log *logger.Logger) *AccrualSystem {
	return &AccrualSystem{accrualSystemAddress: accrualSystemAddress, app: app, log: log}
}

func (accrualSystem *AccrualSystem) AccrualSystemClient(orderNumber string) (models.OrderAccrualInfo, error) {
	client := &http.Client{}
	urlAccSys, err := url.JoinPath(accrualSystem.accrualSystemAddress, "api/orders/", orderNumber)
	if err != nil {
		accrualSystem.log.Sugar().Errorf("Failed to join provided URL path for Accrual System: %s", err)
	}

	response, err := client.Get(urlAccSys)
	if err != nil {
		return models.OrderAccrualInfo{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNoContent:
		return models.OrderAccrualInfo{}, ErrNoContent
	case http.StatusTooManyRequests:
		return models.OrderAccrualInfo{}, ErrAccSysTooManyRequests
	case http.StatusInternalServerError:
		return models.OrderAccrualInfo{}, ErrAccSysInternalServerError
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		accrualSystem.log.Sugar().Errorf("Failed to read response body: %s", err)
		return models.OrderAccrualInfo{}, ErrAccSysInternalServerError
	}

	var orderAccrualInfo models.OrderAccrualInfo
	err = json.Unmarshal(body, &orderAccrualInfo)
	if err != nil {
		accrualSystem.log.Sugar().Errorf("Failed to unmarshal response body: %s", err)
		return models.OrderAccrualInfo{}, ErrAccSysInternalServerError
	}

	return orderAccrualInfo, nil

}

func Run(accrualSystem *AccrualSystem) {
	accrualSystem.log.Sugar().Infof("Running server on %s", accrualSystem.accrualSystemAddress)

	orderNumbersChannel := make(chan string, 10)
	defer close(orderNumbersChannel)

	orderAccrualInfoChannel := make(chan models.OrderAccrualInfo, 10)
	defer close(orderAccrualInfoChannel)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			<-ticker.C
			go accrualSystem.app.GetAccrualOrdersNumbersInfo(orderNumbersChannel)
		}
	}()

	go accrualSystem.app.UpdateAccrualInfo(orderAccrualInfoChannel)

	for orderNumber := range orderNumbersChannel {
		orderAccrualInfo, err := accrualSystem.AccrualSystemClient(orderNumber)
		if err != nil {
			switch err {
			case ErrNoContent:
				accrualSystem.log.Sugar().Errorf(err.Error())
			case ErrAccSysTooManyRequests:
				accrualSystem.log.Sugar().Errorf(err.Error())
				time.Sleep(time.Second)
			case ErrAccSysInternalServerError:
				accrualSystem.log.Sugar().Errorf(err.Error())
			}
		}
		orderAccrualInfoChannel <- orderAccrualInfo
	}

}
