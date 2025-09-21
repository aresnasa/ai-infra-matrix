package services

import (
    "bytes"
    "context"
    "encoding/base64"
    "fmt"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
    "gorm.io/gorm"
)

// FilesService 提供远程文件浏览与上传下载能力（通过 SSH）
type FilesService struct {
    db      *gorm.DB
    sshSvc  *SSHService
}

func NewFilesService(db *gorm.DB, sshSvc *SSHService) *FilesService {
    return &FilesService{db: db, sshSvc: sshSvc}
}

// ListDirectory 列出远程目录内容
func (fs *FilesService) ListDirectory(ctx context.Context, clusterID, dir string) ([]models.FileInfo, error) {
    if dir == "" {
        dir = "/home"
    }

    // 查询集群信息以建立 SSH 连接
    var cluster models.Cluster
    if err := fs.db.Where("id = ? AND status = 'active'", clusterID).First(&cluster).Error; err != nil {
        return nil, fmt.Errorf("cluster not found: %w", err)
    }

    // 使用 ls -la --time-style=+%s 统一时间格式（秒时间戳）
    cmd := fmt.Sprintf("/bin/sh -lc 'ls -la --time-style=+%s %s'", "%s", escapePath(dir))
    out, err := fs.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
    if err != nil {
        return nil, fmt.Errorf("list directory failed: %w", err)
    }

    // 解析输出
    lines := strings.Split(out, "\n")
    var files []models.FileInfo
    now := time.Now()
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "total ") {
            continue
        }
        // 格式类似: -rw-r--r-- 1 root root 4096 1700000000 filename
        parts := strings.Fields(line)
        if len(parts) < 7 { // 权限、链接数、用户、组、大小、时间戳、名称
            continue
        }
        mode := parts[0]
        sizeStr := parts[4]
        tsStr := parts[5]
        name := strings.Join(parts[6:], " ")
        if name == "." || name == ".." {
            continue
        }
        // 解析大小
        var size int64
        fmt.Sscan(sizeStr, &size)
        // 解析时间戳
        var ts int64
        fmt.Sscan(tsStr, &ts)
        modTime := time.Unix(ts, 0)
        // 是否目录
        isDir := strings.HasPrefix(mode, "d")
        files = append(files, models.FileInfo{
            Name:    name,
            Path:    filepath.Join(dir, name),
            Size:    size,
            IsDir:   isDir,
            ModTime: modTime,
            Mode:    mode,
        })
    }

    // 排序：目录优先，其次名称
    sort.Slice(files, func(i, j int) bool {
        if files[i].IsDir != files[j].IsDir {
            return files[i].IsDir && !files[j].IsDir
        }
        return files[i].Name < files[j].Name
    })

    // 如果目录为空，返回空数组而非错误
    _ = now // 保留引用避免未使用告警（兼容某些编译器设置）
    return files, nil
}

// DownloadFile 读取远程文件内容
func (fs *FilesService) DownloadFile(ctx context.Context, clusterID, filePath string) ([]byte, error) {
    var cluster models.Cluster
    if err := fs.db.Where("id = ? AND status = 'active'", clusterID).First(&cluster).Error; err != nil {
        return nil, fmt.Errorf("cluster not found: %w", err)
    }

    // 使用 base64 进行安全传输，避免编码问题
    cmd := fmt.Sprintf("/bin/sh -lc 'base64 -w0 %s'", escapePath(filePath))
    out, err := fs.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
    if err != nil {
        return nil, fmt.Errorf("read remote file failed: %w", err)
    }
    data, decErr := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
    if decErr != nil {
        return nil, fmt.Errorf("decode base64 failed: %w", decErr)
    }
    return data, nil
}

// UploadFile 将本地字节内容写入远程文件（使用 base64 安全传输）
func (fs *FilesService) UploadFile(ctx context.Context, clusterID, filePath string, content []byte) error {
    var cluster models.Cluster
    if err := fs.db.Where("id = ? AND status = 'active'", clusterID).First(&cluster).Error; err != nil {
        return fmt.Errorf("cluster not found: %w", err)
    }

    // 编码为 base64，远程解码
    b64 := base64.StdEncoding.EncodeToString(content)
    // 分块写入避免命令长度限制
    const chunkSize = 32 * 1024 // 32KB
    var buf bytes.Buffer
    for i := 0; i < len(b64); i += chunkSize {
        end := i + chunkSize
        if end > len(b64) {
            end = len(b64)
        }
        chunk := b64[i:end]
        // 逐块追加到临时文件
        appendCmd := fmt.Sprintf("/bin/sh -lc 'printf %s >> /tmp/.aimatrix_upload.b64'", singleQuoted(chunk))
        if _, err := fs.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", appendCmd); err != nil {
            return fmt.Errorf("upload chunk failed: %w", err)
        }
    }
    _ = buf

    // 解码到目标路径并清理临时文件
    finalize := fmt.Sprintf("/bin/sh -lc 'base64 -d /tmp/.aimatrix_upload.b64 > %s && rm -f /tmp/.aimatrix_upload.b64'", escapePath(filePath))
    if _, err := fs.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", finalize); err != nil {
        return fmt.Errorf("finalize upload failed: %w", err)
    }
    return nil
}

// 工具：对路径进行安全转义（简单场景）
func escapePath(p string) string {
    if p == "" { return p }
    // 用单引号包裹，并将内部单引号替换为'\''
    return "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
}

// 工具：将任意字符串安全作为 printf 参数
func singleQuoted(s string) string {
    return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
