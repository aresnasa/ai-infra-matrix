package controllers

import (
    "net/http"
    "io"
    "bytes"
    "encoding/json"
    "github.com/gin-gonic/gin"
)

// MinIOConsoleLoginRequest represents the input payload for server-side login
type MinIOConsoleLoginRequest struct {
    AccessKey string `json:"access_key" binding:"required"`
    SecretKey string `json:"secret_key" binding:"required"`
}

// Proxy login to MinIO Console via server to set cookies for same-origin iframe
func MinIOConsoleProxyLogin(c *gin.Context) {
    var req MinIOConsoleLoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    // Try multiple endpoint variants
    endpoints := []string{
        "http://nginx/minio-console/api/v1/login",
        "http://nginx/minio-console/api/login",
    }

    payloads := []map[string]string{
        {"username": req.AccessKey, "password": req.SecretKey},
        {"accessKey": req.AccessKey, "secretKey": req.SecretKey},
    }

    var lastStatus int = http.StatusBadGateway
    for _, ep := range endpoints {
        for _, p := range payloads {
            b, _ := json.Marshal(p)
            r, err := http.NewRequest("POST", ep, bytes.NewReader(b))
            if err != nil {
                continue
            }
            r.Header.Set("Content-Type", "application/json")
            r.Header.Set("Accept", "application/json")
            r.Header.Set("X-Requested-With", "XMLHttpRequest")

            resp, err := http.DefaultClient.Do(r)
            if err != nil {
                continue
            }
            defer resp.Body.Close()
            lastStatus = resp.StatusCode

            // proxy set-cookie headers to client
            for k, vals := range resp.Header {
                if k == "Set-Cookie" {
                    for _, v := range vals {
                        c.Writer.Header().Add("Set-Cookie", v)
                    }
                }
            }

            body, _ := io.ReadAll(resp.Body)
            if resp.StatusCode >= 200 && resp.StatusCode < 300 {
                c.Data(http.StatusOK, resp.Header.Get("Content-Type"), body)
                return
            }
        }
    }

    c.JSON(lastStatus, gin.H{"error": "minio console login failed"})
}
