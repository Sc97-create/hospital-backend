package config

import (
	"log"
)

func Init() {
	postgreStr := "host=localhost user=postgres password=Sachin@1998 dbname=hospitaldb_local port=5469 sslmode=disable"
	err := PostgreClient.CreateClient(postgreStr)
	if err != nil {
		log.Fatal("%v", err)
	}
}
