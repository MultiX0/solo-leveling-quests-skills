package main

import (
	"log"

	"github.com/MultiX0/solo_leveling_system/api"
	"github.com/MultiX0/solo_leveling_system/db"
	"github.com/MultiX0/solo_leveling_system/jobs"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db.InitDB()
	jobs.InitCronJobs()

	server := api.NewServer(":8080")
	server.RunServer()
}
