package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/DariSorokina/yp-gophermart.git/internal/models"
	"github.com/DariSorokina/yp-gophermart.git/internal/utils"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
)

const (
	registerQuery               = `INSERT INTO content.users (user_login, encrypted_password) VALUES ($1, $2) RETURNING user_id;`
	loginQuery                  = `SELECT user_id, encrypted_password FROM content.users WHERE user_login = $1;`
	postOrderNumberQuery        = `INSERT INTO content.orders (user_id, order_id, accrual, status, flag) VALUES ($1, $2, 0, 'NEW', 'accrue');`
	getUserIDQuery              = `SELECT user_id FROM content.orders WHERE order_id = $1`
	getOrdersNumbersQuery       = `SELECT order_id, COALESCE(accrual, 0) AS accrual, status, uploaded_at FROM content.orders WHERE user_id = $1 ORDER BY uploaded_at ASC;`
	getLoyaltyBalanceQuery      = `SELECT accrual FROM content.orders WHERE user_id = $1;`
	withdrawLoyaltyBonusesQuery = `INSERT INTO content.orders (user_id, order_id, accrual, flag) VALUES ($1, $2, $3, 'withdraw');`
	withdrawalsInfoQuery        = `SELECT order_id, accrual, uploaded_at FROM content.orders WHERE user_id = $1 AND accrual < 0 ORDER BY uploaded_at ASC;`
	updateAccuralQuery          = `UPDATE content.orders SET accrual = $1, status = $2 WHERE order_id = $3 AND flag = 'accrue';`
	getAccrualStatusQuery       = `SELECT order_id FROM content.orders WHERE status = 'NEW' OR status = 'PROCESSING';`
)

var (
	ErrNotEnoughLoyaltyBonuses = errors.New("postgresql: not enough loyalty bonuses")
)

type PostgresqlDB struct {
	db  *sql.DB
	log *logger.Logger
}

func NewPostgresqlDB(cofigBDString string, l *logger.Logger) (*PostgresqlDB, error) {
	db, err := sql.Open("pgx", cofigBDString)
	if err != nil {
		l.Sugar().Errorf("Failed to open a database: %s", err)
		return &PostgresqlDB{db: db, log: l}, err
	}

	filePath := utils.GetCurrentFilePath()
	parentDir := utils.GetParentDirectory(filePath)
	pathToMigrations := parentDir + "/migrations"

	err = goose.Up(db, pathToMigrations)
	if err != nil {
		l.Sugar().Fatalf("goose.Up: %v", err)
	}

	return &PostgresqlDB{db: db, log: l}, nil
}

func (postgresqlDB *PostgresqlDB) Register(ctx context.Context, registerRequest models.RegisterRequest) (userID int, err error) {

	encryptedPassword := utils.HashPassword(registerRequest.Password)
	err = postgresqlDB.db.QueryRowContext(ctx, registerQuery, registerRequest.Login, encryptedPassword).Scan(&userID)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query registerQuery: %s", err)
		return userID, err
	}

	return userID, nil
}

func (postgresqlDB *PostgresqlDB) Login(ctx context.Context, registerRequest models.RegisterRequest) (userID int, err error) {

	var encryptedPassword string
	err = postgresqlDB.db.QueryRowContext(ctx, loginQuery, registerRequest.Login).Scan(&userID, &encryptedPassword)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query loginQuery: %s", err)
		return userID, err
	}

	err = utils.CheckPassword(encryptedPassword, registerRequest.Password)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf(err.Error())
		return userID, err
	}

	return userID, nil
}

func (postgresqlDB *PostgresqlDB) PostOrderNumber(ctx context.Context, userID int, orderNumber string) (retrievedUserID int, err error) {

	result, err := postgresqlDB.db.ExecContext(ctx, postOrderNumberQuery, userID, orderNumber)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query postOrderNumberQuery: %s", err)
		log.Println(err.Error())

		var pgError *pgconn.PgError
		var pgUserID int

		if ok := errors.As(err, &pgError); ok && pgError.Code == pgerrcode.UniqueViolation {
			err = postgresqlDB.db.QueryRowContext(ctx, getUserIDQuery, orderNumber).Scan(&pgUserID)
			if err != nil {
				postgresqlDB.log.Sugar().Errorf("Failed to execute a query getUserIDQuery: %s", err)
				return retrievedUserID, err
			}
			retrievedUserID = pgUserID
		}
		return retrievedUserID, nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute RowsAffected: %s", err)
		postgresqlDB.log.Sugar().Infof("Affected rows: %d", rows)
		log.Println(err.Error())
		return userID, err
	}
	return retrievedUserID, err
}

func (postgresqlDB *PostgresqlDB) GetOrdersNumbers(ctx context.Context, userID int) (orders []models.OrderInfo, err error) {
	rows, err := postgresqlDB.db.QueryContext(ctx, getOrdersNumbersQuery, userID)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query getOrdersNumbersQuery: %s", err)
		return orders, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.OrderInfo
		if err := rows.Scan(&order.OrderNumber, &order.Accrual, &order.Status, &order.UploadedAt); err != nil {
			postgresqlDB.log.Sugar().Errorf("Failed to scan order information in GetOrdersNumbers method: %s", err)
		}
		orders = append(orders, order)
	}

	rerr := rows.Close()
	if rerr != nil {
		postgresqlDB.log.Sugar().Errorf("Close error in GetOrdersNumbers method: %s", rerr)
		return orders, rerr
	}

	if err := rows.Err(); err != nil {
		postgresqlDB.log.Sugar().Errorf("The last error encountered by Rows.Scan in GetOrdersNumbers method: %s", err)
		return orders, err
	}

	return orders, err
}

func (postgresqlDB *PostgresqlDB) GetLoyaltyBalance(ctx context.Context, tx *sql.Tx, userID int) (balance models.BalanceInfo, err error) {

	var rows *sql.Rows

	if tx != nil {
		rows, err = tx.QueryContext(ctx, getLoyaltyBalanceQuery, userID)
	} else {
		rows, err = postgresqlDB.db.QueryContext(ctx, getLoyaltyBalanceQuery, userID)
	}

	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query getLoyaltyBalanceQuery: %s", err)
		return balance, err
	}
	defer rows.Close()

	var currentBalance float64
	var withdrawn float64

	for rows.Next() {
		var accuralValue float64
		if err := rows.Scan(&accuralValue); err != nil {
			postgresqlDB.log.Sugar().Errorf("Failed to scan order information in GetLoyaltyBalance method: %s", err)
		}
		if accuralValue < 0 {
			withdrawn += accuralValue
		}
		currentBalance += accuralValue
	}

	balance.CurrentBalance = currentBalance
	switch withdrawn {
	case 0:
		balance.Withdrawn = withdrawn
	default:
		balance.Withdrawn = withdrawn * (-1)
	}

	rerr := rows.Close()
	if rerr != nil {
		postgresqlDB.log.Sugar().Errorf("Close error in GetLoyaltyBalance method: %s", rerr)
		return balance, rerr
	}

	if err := rows.Err(); err != nil {
		postgresqlDB.log.Sugar().Errorf("The last error encountered by Rows.Scan in GetLoyaltyBalance method: %s", err)
		return balance, err
	}

	return balance, err
}

func (postgresqlDB *PostgresqlDB) WithdrawLoyaltyBonuses(ctx context.Context, userID int, withdrawRequest models.WithdrawRequest) (err error) {

	tx, err := postgresqlDB.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	balance, err := postgresqlDB.GetLoyaltyBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	if balance.CurrentBalance < withdrawRequest.Sum {
		return ErrNotEnoughLoyaltyBonuses
	}

	result, err := tx.ExecContext(ctx, withdrawLoyaltyBonusesQuery, userID, withdrawRequest.OrderNumber, withdrawRequest.Sum*(-1))
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query withdrawLoyaltyBonusesQuery: %s", err)
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute RowsAffected: %s", err)
		postgresqlDB.log.Sugar().Infof("Affected rows: %d", rows)
		log.Println(err.Error())
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (postgresqlDB *PostgresqlDB) GetWithdrawalsInfo(ctx context.Context, userID int) (orders []models.WithdrawalInfo, err error) {
	rows, err := postgresqlDB.db.QueryContext(ctx, withdrawalsInfoQuery, userID)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query withdrawalsInfoQuery: %s", err)
		return orders, err
	}
	defer rows.Close()

	var order models.WithdrawalInfo
	for rows.Next() {
		if err := rows.Scan(&order.OrderNumber, &order.Sum, &order.ProcessedAt); err != nil {
			postgresqlDB.log.Sugar().Errorf("Failed to scan order information in GetWithdrawalsInfo method: %s", err)
		}
		order.Sum *= -1
		orders = append(orders, order)
	}

	rerr := rows.Close()
	if rerr != nil {
		postgresqlDB.log.Sugar().Errorf("Close error in GetWithdrawalsInfo method: %s", rerr)
	}

	if err := rows.Err(); err != nil {
		postgresqlDB.log.Sugar().Errorf("The last error encountered by Rows.Scan in GetWithdrawalsInfo method: %s", err)
		log.Fatal(err)
	}

	return
}

func (postgresqlDB *PostgresqlDB) UpdateAccrualInfo(orderAccrualInfo models.OrderAccrualInfo) {

	result, err := postgresqlDB.db.Exec(updateAccuralQuery, orderAccrualInfo.Accrual, orderAccrualInfo.Status, orderAccrualInfo.OrderNumber)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query updateAccuralQuery: %s", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute RowsAffected: %s", err)
		postgresqlDB.log.Sugar().Infof("Affected rows: %d", rows)
	}

}

func (postgresqlDB *PostgresqlDB) GetAccrualOrdersNumbersInfo(orderNumbersChannel chan string) {

	rows, err := postgresqlDB.db.Query(getAccrualStatusQuery)
	if err != nil {
		postgresqlDB.log.Sugar().Errorf("Failed to execute a query getAccrualStatusQuery: %s", err)
	}
	defer rows.Close()

	var order string
	for rows.Next() {
		if err := rows.Scan(&order); err != nil {
			postgresqlDB.log.Sugar().Errorf("Failed to scan order information in GetAccrualOrdersNumbersInfo method: %s", err)
		}
		orderNumbersChannel <- order
	}

	rerr := rows.Close()
	if rerr != nil {
		postgresqlDB.log.Sugar().Errorf("Close error in GetAccrualOrdersNumbersInfo method: %s", rerr)
	}

	if err := rows.Err(); err != nil {
		postgresqlDB.log.Sugar().Errorf("The last error encountered by Rows.Scan in GetAccrualOrdersNumbersInfo method: %s", err)
		log.Fatal(err)
	}

}

func (postgresqlDB *PostgresqlDB) Close() {
	if postgresqlDB.db != nil {
		postgresqlDB.db.Close()
	}
}
