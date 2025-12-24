package models

import "time"

type Log struct {
	Message      string    `bson:"message" json:"message"`
	IpAddress    string    `bson:"ip_address" json:"ip_address"` // Actual IP
	CustomerId   int       `bson:"customer_id" json:"customer_id"`
	LogLevelId   int       `bson:"log_level_id" json:"log_level_id"`
	CreatedOnUtc time.Time `bson:"created_on_utc" json:"created_on_utc"`
}
