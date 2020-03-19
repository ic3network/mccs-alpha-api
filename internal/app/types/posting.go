package types

import (
	"github.com/jinzhu/gorm"
)

type Posting struct {
	gorm.Model
	AccountNumber uint    `gorm:"not null"`
	JournalID     uint    `gorm:"not null"`
	Amount        float64 `gorm:"not null"`
}
