package models

import "time"

type Status string

const (
	New        Status = "NEW"        // Storage
	Processing Status = "PROCESSING" // Accrual System, Storage
	Invalid    Status = "INVALID"    // Accrual System, Storage
	Processed  Status = "PROCESSED"  // Accrual System, Storage
	Registered Status = "REGISTERED" // Accrual System
)

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type OrderInfo struct {
	OrderNumber string    `json:"number"`
	Status      Status    `json:"status"`
	Accrual     float64   `json:"accrual,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type OrderAccrualInfo struct {
	OrderNumber string  `json:"order"`
	Status      Status  `json:"status"`
	Accrual     float64 `json:"accrual,omitempty"`
}

type WithdrawalInfo struct {
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type BalanceInfo struct {
	CurrentBalance float64 `json:"current"`
	Withdrawn      float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
}
