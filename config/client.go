package config

import (
	"log"
	"minio-demo/global"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func InitMinio() {
	endpoint := "111.229.31.132:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"

	clinet, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})

	if err != nil {
		log.Fatalln(err)
		return
	}
	global.GAV_MINIO = clinet
	log.Printf("%#v\n", clinet)
}
