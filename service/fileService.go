package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"minio-demo/dto"
	"minio-demo/global"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

const BucketName = "test"

// 普通文件上传
func Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("上传文件失败: %s", err.Error())})
		return
	}
	objectName := file.Filename // 存储在MinIO中的对象名称

	fileReader, err := file.Open()
	if err != nil {
		// 处理错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fileReader.Close()
	minioClient := global.GAV_MINIO
	uploadInfo, err := minioClient.PutObject(context.Background(), BucketName, objectName, fileReader, file.Size, minio.PutObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("putObject-result:{}", uploadInfo)
	c.JSON(http.StatusOK, gin.H{"message": "File(s) uploaded successfully"})
}

// 通过文件名称模糊匹配文件信息
func GetFile(c *gin.Context) {
	fileName := c.Query("fileName")

	minioClient := global.GAV_MINIO

	objects := minioClient.ListObjects(context.Background(), BucketName, minio.ListObjectsOptions{
		Prefix: fileName,
	})

	var result string

	for object := range objects {
		if object.Err != nil {
			result = object.Err.Error()
			break
		}
		log.Printf("object:{}", object)
		result += object.Key + ";"
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// 分片文件上传
func UploadPart(c *gin.Context) {
	var params dto.FilePartUploadDTO
	err := c.ShouldBind(&params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("分片上传入参:{}", params)

	fileReader, err := file.Open()
	if err != nil {
		// 处理错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fileReader.Close()
	result, err := uploadPartCore(fileReader, file.Size, params.Md5, params.Index, params.TotalPart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if result == -1 {
		//表示已经上传结束，进行文件合并，并且删除临时分片文件
		go composeAndDeletePart(params.FileName, params.TotalPart, params.Md5)
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 上传分片文件
// part 分片文件
// partSize 分片大小
// md5 文件MD5
// index 分片序号
// totalPart 总分片数
func uploadPartCore(part multipart.File, partSize int64, md5 string, index int, totalPart int) (int, error) {
	partFileName := "parts/" + md5 + strconv.Itoa(index)
	minioClient := global.GAV_MINIO

	data := minioClient.ListObjects(context.Background(), BucketName, minio.ListObjectsOptions{
		Prefix: "parts/" + md5,
	})
	partNames := make([]string, 0)
	for fileObject := range data {
		partNames = append(partNames, fileObject.Key)
	}

	//判断当前分片是否已经上传过了，同时判断分片是否已经全部上传完了
	if Contains(partNames, partFileName) {
		if len(partNames) == totalPart {
			return -1, nil
		}
		return index, nil
	}

	_, err := minioClient.PutObject(context.Background(), BucketName, partFileName, part, partSize, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}

	//判断当前要上传的分片，是否是最后一个
	if len(partNames)+1 == totalPart {
		return -1, nil
	}
	return index, nil
}

// 合并分片并删除分片
func composeAndDeletePart(fileName string, totalPart int, md5 string) error {
	minioClient := global.GAV_MINIO
	destOptions := minio.CopyDestOptions{
		Bucket: BucketName,
		Object: "parts/" + fileName,
	}
	srcs := make([]minio.CopySrcOptions, 0)
	for i := 1; i <= totalPart; i++ {
		src := minio.CopySrcOptions{
			Bucket: BucketName,
			Object: "parts/" + md5 + strconv.Itoa(i),
		}
		srcs = append(srcs, src)
	}

	//合并分片
	_, err := minioClient.ComposeObject(context.Background(), destOptions, srcs...)

	return err
}

func Contains(slice []string, element string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}

// 大文件拆分切片，存储到本地临时目录
func SpiltBigFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//获取文件md5，作为分片的文件名前缀
	hash := md5.Sum([]byte(file.Filename))
	partNamePrefix := hex.EncodeToString(hash[:])

	partMaxSize := 10 * 1024 * 1024 // 每个分片的最大值（这里设定为 10MB）
	partMinSize := 5 * 1024 * 1024  // 每个分片的最小值（这里设定为 5MB）

	size := file.Size
	partNum := int(size) / partMaxSize      //分片数
	lastPartSize := int(size) % partMaxSize //最后一个分片的大小

	if lastPartSize <= partMinSize {
		//最后一个分片小于5M的时候，合并到前一个分片
		partNum = partNum - 1
		lastPartSize = lastPartSize + partMaxSize
	}

	buffer := make([]byte, partMaxSize)

	fileReader, err := file.Open()
	if err != nil {
		// 处理错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fileReader.Close()

	filePath := "D:\\Program Files\\ffmpeg-6.0\\projcets\\parts\\"
	for index := 1; index <= partNum; index++ {
		if index == partNum {
			//最后一个分片
			buffer = make([]byte, lastPartSize)
		}
		n, err := fileReader.Read(buffer)

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if n == 0 {
			break
		}

		partFileName := filePath + partNamePrefix + strconv.Itoa(index)
		outputFile, err := os.Create(partFileName)
		if err != nil {
			log.Fatal(err)
		}
		defer outputFile.Close()
		_, err = outputFile.Write(buffer[:n])
		if err != nil {
			log.Fatal(err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
