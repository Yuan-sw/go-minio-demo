package dto

// 文件分片上传入参结构体
type FilePartUploadDTO struct {
	Index     int    `form:"index" json:"index" binding:"required" error:"分片序号不能为空"`
	TotalPart int    `form:"totalPart" json:"totalPart" binding:"required" error:"总分片数不能为空"`
	Md5       string `form:"md5" json:"md5" binding:"required" error:"文件唯一编码不能为空"`
	FileName  string `form:"fileName" json:"fileName" binding:"required" error:"文件名称不能为空"`
}
