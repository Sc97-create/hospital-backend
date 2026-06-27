package database

import (
	"hospital-backend/pkg/db"
)

var (
	PostgreClient db.Postgre
)

func Connect(databaseURl string) error {
	err := PostgreClient.CreateClient(databaseURl)
	if err != nil {
		return err
	}
	return nil
}
