package controllers

import (
    "io"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
)

// FilesController 远程文件浏览/传输控制器
type FilesController struct {
    filesSvc *services.FilesService
}

func NewFilesController(filesSvc *services.FilesService) *FilesController {
    return &FilesController{filesSvc: filesSvc}
}

// List 列出目录
// GET /api/files?cluster=CLUSTER_ID&path=/path
func (fc *FilesController) List(c *gin.Context) {
    clusterID := c.Query("cluster")
    path := c.Query("path")
    if clusterID == "" {
        c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "cluster 参数必填"})
        return
    }
    files, err := fc.filesSvc.ListDirectory(c.Request.Context(), clusterID, path)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "获取目录失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, models.Response{Code: 200, Message: "success", Data: files})
}

// Download 下载文件
// GET /api/files/download?cluster=CLUSTER_ID&path=/abs/file
func (fc *FilesController) Download(c *gin.Context) {
    clusterID := c.Query("cluster")
    path := c.Query("path")
    if clusterID == "" || path == "" {
        c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "cluster 与 path 参数必填"})
        return
    }
    data, err := fc.filesSvc.DownloadFile(c.Request.Context(), clusterID, path)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "下载失败: " + err.Error()})
        return
    }
    c.Header("Content-Disposition", "attachment; filename=\""+ sanitizeFileName(path) +"\"")
    c.Data(http.StatusOK, "application/octet-stream", data)
}

// Upload 上传文件（multipart form）
// POST /api/files/upload (fields: cluster, path, file)
func (fc *FilesController) Upload(c *gin.Context) {
    clusterID := c.PostForm("cluster")
    path := c.PostForm("path")
    file, err := c.FormFile("file")
    if clusterID == "" || path == "" || err != nil {
        c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "缺少必要参数或文件"})
        return
    }
    f, err := file.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "读取文件失败: " + err.Error()})
        return
    }
    defer f.Close()
    buf, err := io.ReadAll(f)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "加载文件失败: " + err.Error()})
        return
    }
    if err := fc.filesSvc.UploadFile(c.Request.Context(), clusterID, path, buf); err != nil {
        c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "上传失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, models.Response{Code: 200, Message: "上传成功"})
}

// sanitizeFileName 简单获取路径末尾名称
func sanitizeFileName(path string) string {
    // 简化：仅取最后一段
    for i := len(path)-1; i >= 0; i-- {
        if path[i] == '/' {
            return path[i+1:]
        }
    }
    return path
}
