package types

import (
	"time"

	"github.com/ic3network/mccs-alpha-api/util"
)

func NewUserRespond(user *User) *UserRespond {
	return &UserRespond{
		ID:                            user.ID.Hex(),
		Email:                         user.Email,
		UserPhone:                     user.Telephone,
		FirstName:                     user.FirstName,
		LastName:                      user.LastName,
		LastLoginIP:                   user.LastLoginIP,
		LastLoginDate:                 user.LastLoginDate,
		DailyEmailMatchNotification:   util.ToBool(user.DailyNotification),
		ShowTagsMatchedSinceLastLogin: util.ToBool(user.ShowRecentMatchedTags),
	}
}

type UserRespond struct {
	ID                            string    `json:"id"`
	Email                         string    `json:"email"`
	FirstName                     string    `json:"firstName"`
	LastName                      string    `json:"lastName"`
	UserPhone                     string    `json:"userPhone"`
	LastLoginIP                   string    `json:"lastLoginIP"`
	LastLoginDate                 time.Time `json:"lastLoginDate"`
	DailyEmailMatchNotification   bool      `json:"dailyEmailMatchNotification"`
	ShowTagsMatchedSinceLastLogin bool      `json:"showTagsMatchedSinceLastLogin"`
}

func NewEntityRespond(entity *Entity) *EntityRespond {
	return &EntityRespond{
		ID:                 entity.ID.Hex(),
		AccountNumber:      entity.AccountNumber,
		EntityName:         entity.EntityName,
		EntityPhone:        entity.EntityPhone,
		IncType:            entity.IncType,
		CompanyNumber:      entity.CompanyNumber,
		Website:            entity.Website,
		Turnover:           entity.Turnover,
		Description:        entity.Description,
		LocationAddress:    entity.LocationAddress,
		LocationCity:       entity.LocationCity,
		LocationRegion:     entity.LocationRegion,
		LocationPostalCode: entity.LocationPostalCode,
		LocationCountry:    entity.LocationCountry,
		Status:             entity.Status,
		Offers:             TagFieldToNames(entity.Offers),
		Wants:              TagFieldToNames(entity.Wants),
	}
}

type EntityRespond struct {
	ID                 string   `json:"id"`
	AccountNumber      string   `json:"accountNumber"`
	EntityName         string   `json:"entityName"`
	EntityPhone        string   `json:"entityPhone"`
	IncType            string   `json:"incType"`
	CompanyNumber      string   `json:"companyNumber"`
	Website            string   `json:"website"`
	Turnover           int      `json:"turnover"`
	Description        string   `json:"description"`
	LocationAddress    string   `json:"locationAddress"`
	LocationCity       string   `json:"locationCity"`
	LocationRegion     string   `json:"locationRegion"`
	LocationPostalCode string   `json:"locationPostalCode"`
	LocationCountry    string   `json:"locationCountry"`
	Status             string   `json:"status"`
	Offers             []string `json:"offers"`
	Wants              []string `json:"wants"`
	IsFavorite         bool     `json:"isFavorite"`
}