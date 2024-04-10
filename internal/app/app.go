package app

import (
	"context"

	"github.com/DariSorokina/yp-gophermart.git/internal/database"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/DariSorokina/yp-gophermart.git/internal/models"
)

type App struct {
	db  *database.PostgresqlDB
	log *logger.Logger
}

func NewApp(db *database.PostgresqlDB, log *logger.Logger) *App {
	return &App{db: db, log: log}
}

func (app *App) Register(ctx context.Context, registerRequest models.RegisterRequest) (userID int, err error) {
	userID, err = app.db.Register(ctx, registerRequest)
	return
}

func (app *App) Login(ctx context.Context, registerRequest models.RegisterRequest) (userID int, err error) {
	userID, err = app.db.Login(ctx, registerRequest)
	return
}

func (app *App) PostOrderNumber(ctx context.Context, userID int, orderNumber string) (retrievedUserID int, err error) {
	retrievedUserID, err = app.db.PostOrderNumber(ctx, userID, orderNumber)
	return
}

func (app *App) GetOrdersNumbers(ctx context.Context, userID int) (orders []models.OrderInfo, err error) {
	orders, err = app.db.GetOrdersNumbers(ctx, userID)
	return
}

func (app *App) GetLoyaltyBalance(ctx context.Context, userID int) (balance models.BalanceInfo, err error) {
	balance, err = app.db.GetLoyaltyBalance(ctx, nil, userID)
	return
}

func (app *App) WithdrawLoyaltyBonuses(ctx context.Context, userID int, withdrawRequest models.WithdrawRequest) (err error) {
	err = app.db.WithdrawLoyaltyBonuses(ctx, userID, withdrawRequest)
	return err
}

func (app *App) GetWithdrawalsInfo(ctx context.Context, userID int) (orders []models.WithdrawalInfo, err error) {
	orders, err = app.db.GetWithdrawalsInfo(ctx, userID)
	return
}

// accrual system
func (app *App) UpdateAccrualInfo(orderAccrualInfoChannel chan models.OrderAccrualInfo) {
	for orderAccrualInfo := range orderAccrualInfoChannel {
		app.db.UpdateAccrualInfo(orderAccrualInfo)
	}
}

func (app *App) GetAccrualOrdersNumbersInfo(orderNumbersChannel chan string) {
	app.db.GetAccrualOrdersNumbersInfo(orderNumbersChannel)
}
