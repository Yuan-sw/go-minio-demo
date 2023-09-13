package main

import (
	"log"
	"minio-demo/config"
)

func main() {
	config.InitMinio()
	router := config.Routers()
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
