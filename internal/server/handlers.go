package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/DariSorokina/yp-gophermart.git/internal/app"
	"github.com/DariSorokina/yp-gophermart.git/internal/config"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/DariSorokina/yp-gophermart.git/internal/models"
	"github.com/DariSorokina/yp-gophermart.git/internal/utils"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotEnoughLoyaltyBonuses = errors.New("postgresql: not enough loyalty bonuses")
)

type handlers struct {
	app        *app.App
	flagConfig *config.FlagConfig
	log        *logger.Logger
}

func newHandlers(app *app.App, flagConfig *config.FlagConfig, l *logger.Logger) *handlers {
	return &handlers{app: app, flagConfig: flagConfig, log: l}
}

func (handlers *handlers) registerHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	var registerRequest models.RegisterRequest

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(requestBody, &registerRequest); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if registerRequest.Login == "" || registerRequest.Password == "" {
		http.Error(res, "Missing login or password", http.StatusBadRequest)
		return
	}

	var pgError *pgconn.PgError

	userID, err := handlers.app.Register(ctx, registerRequest)
	if err != nil {
		if ok := errors.As(err, &pgError); ok && pgError.Code == pgerrcode.UniqueViolation {
			res.WriteHeader(http.StatusConflict)
		}
		res.WriteHeader(http.StatusInternalServerError)
	}
	userIDstring := strconv.Itoa(userID)
	res.Header().Add("ClientID", userIDstring)

}

func (handlers *handlers) loginHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	var registerRequest models.RegisterRequest

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(requestBody, &registerRequest); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if registerRequest.Login == "" || registerRequest.Password == "" {
		http.Error(res, "Missing login or password", http.StatusBadRequest)
		return
	}

	userID, err := handlers.app.Login(ctx, registerRequest)
	if err != nil {
		switch {
		case err == sql.ErrNoRows || err == bcrypt.ErrMismatchedHashAndPassword:
			res.WriteHeader(http.StatusUnauthorized)
		default:
			res.WriteHeader(http.StatusInternalServerError)
		}

	}

	userIDstring := strconv.Itoa(userID)
	res.Header().Add("ClientID", userIDstring)
}

func (handlers *handlers) postOrderNumberHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	userID := req.Header.Get("ClientID")
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		handlers.log.Sugar().Errorf("Failed to parse client ID: %s", err)
		fmt.Println(userID)
		return
	}

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	orederNumberStr := string(requestBody)
	if !utils.IsValidLuhn(orederNumberStr) {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	var retrievedUserID int
	retrievedUserID, err = handlers.app.PostOrderNumber(ctx, userIDInt, orederNumberStr)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch retrievedUserID {
	case 0:
		res.WriteHeader(http.StatusAccepted)
	case userIDInt:
		res.WriteHeader(http.StatusOK)
	default:
		res.WriteHeader(http.StatusConflict)
	}

}

func (handlers *handlers) getOrdersNumbersHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	userID := req.Header.Get("ClientID")
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		handlers.log.Sugar().Errorf("Failed to parse client ID: %s", err)
		fmt.Println(userID)
		return
	}

	var orders []models.OrderInfo
	orders, err = handlers.app.GetOrdersNumbers(ctx, userIDInt)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	resp, err := json.Marshal(orders)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(resp)

}

func (handlers *handlers) getLoyaltyBalanceHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	userID := req.Header.Get("ClientID")
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		handlers.log.Sugar().Errorf("Failed to parse client ID: %s", err)
		return
	}

	var balance models.BalanceInfo
	balance, err = handlers.app.GetLoyaltyBalance(ctx, userIDInt)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(balance)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(resp)
}

func (handlers *handlers) withdrawLoyaltyBonusesHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	userID := req.Header.Get("ClientID")
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		handlers.log.Sugar().Errorf("Failed to parse client ID: %s", err)
		return
	}

	var withdrawRequest models.WithdrawRequest

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(requestBody, &withdrawRequest); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if !utils.IsValidLuhn(withdrawRequest.OrderNumber) {
		res.WriteHeader(http.StatusUnprocessableEntity)
	}

	err = handlers.app.WithdrawLoyaltyBonuses(ctx, userIDInt, withdrawRequest)

	switch {
	case errors.Is(err, ErrNotEnoughLoyaltyBonuses):
		res.WriteHeader(http.StatusPaymentRequired)
	case err != nil:
		res.WriteHeader(http.StatusInternalServerError)
	default:
		res.WriteHeader(http.StatusOK)
	}
}

func (handlers *handlers) withdrawalsInfoHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	userID := req.Header.Get("ClientID")
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		handlers.log.Sugar().Errorf("Failed to parse client ID: %s", err)
		return
	}

	var orders []models.WithdrawalInfo
	orders, err = handlers.app.GetWithdrawalsInfo(ctx, userIDInt)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	resp, err := json.Marshal(orders)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(resp)
}
