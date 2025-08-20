package services

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
    "github.com/sirupsen/logrus"
)

type GiteaService interface {
    EnsureUser(u models.User) error
    SyncAllUsers() (created, updated, skipped int, err error)
}

type giteaServiceImpl struct {
    cfg *config.Config
    httpClient *http.Client
}

type giteaUserCreateReq struct {
    Username string `json:"username"`
    LoginName string `json:"login_name,omitempty"`
    Email    string `json:"email"`
    FullName string `json:"full_name,omitempty"`
    Password string `json:"password,omitempty"`
    MustChangePassword bool `json:"must_change_password"`
    SendNotify bool `json:"send_notify"`
}

type giteaUserUpdateReq struct {
    LoginName string `json:"login_name,omitempty"`
    Email    string `json:"email,omitempty"`
    FullName string `json:"full_name,omitempty"`
    Active   *bool  `json:"active,omitempty"`
}

type giteaUserResp struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    FullName string `json:"full_name"`
    Active   bool   `json:"active"`
}

func NewGiteaService(cfg *config.Config) GiteaService {
    return &giteaServiceImpl{
        cfg: cfg,
        httpClient: &http.Client{ Timeout: 10 * time.Second },
    }
}

func (s *giteaServiceImpl) api(path string) string {
    return fmt.Sprintf("%s/api/v1%s", s.cfg.Gitea.BaseURL, path)
}

func (s *giteaServiceImpl) auth(req *http.Request) {
    req.Header.Set("Authorization", fmt.Sprintf("token %s", s.cfg.Gitea.AdminToken))
    req.Header.Set("Content-Type", "application/json")
}

// resolveUsername applies aliasing rules for reserved names (e.g., "admin").
func (s *giteaServiceImpl) resolveUsername(u models.User) (username string) {
    username = u.Username
    if username == "admin" {
        alias := s.cfg.Gitea.AliasAdminTo
        if alias != "" {
            username = alias
        }
    }
    return
}

// EnsureUser creates or updates user in Gitea using admin API
func (s *giteaServiceImpl) EnsureUser(u models.User) error {
    if !s.cfg.Gitea.Enabled {
        return nil
    }
    if s.cfg.Gitea.AdminToken == "" {
        return fmt.Errorf("Gitea Admin token not configured")
    }

    // Map reserved usernames if necessary (e.g., admin -> test)
    targetUsername := s.resolveUsername(u)

    // First, try to get user (public endpoint)
    // Note: GET /admin/users/{username} returns 405 in Gitea; use /users/{username} to check existence
    getURL := s.api(fmt.Sprintf("/users/%s", targetUsername))
    req, _ := http.NewRequest("GET", getURL, nil)
    // Public endpoint: do NOT send Authorization to avoid 401 if admin token is invalid
    resp, err := s.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        if !s.cfg.Gitea.AutoUpdate {
            return nil
        }
    // Update user if email/fullname differs
    // We can't easily diff without fetching body; send update to be idempotent
    active := u.IsActive
    payload := giteaUserUpdateReq{ LoginName: targetUsername, Email: u.Email, FullName: targetUsername, Active: &active }
        body, _ := json.Marshal(payload)
    // Admin endpoint required for update
    updURL := s.api(fmt.Sprintf("/admin/users/%s", targetUsername))
    updReq, _ := http.NewRequest("PATCH", updURL, bytes.NewReader(body))
        s.auth(updReq)
        updResp, err := s.httpClient.Do(updReq)
        if err != nil { return err }
        defer updResp.Body.Close()
        if updResp.StatusCode >= 300 { return fmt.Errorf("gitea update failed: %d", updResp.StatusCode) }
        return nil
    }

    if resp.StatusCode == http.StatusNotFound {
        if !s.cfg.Gitea.AutoCreate {
            return nil
        }
        // Create user
        payload := giteaUserCreateReq{
            Username: targetUsername,
            LoginName: targetUsername,
            Email:    u.Email,
        FullName: targetUsername,
            Password: randomPassword(), // temporary, user logs via reverse-proxy anyway
            MustChangePassword: false,
            SendNotify: false,
        }
        body, _ := json.Marshal(payload)
        createURL := s.api("/admin/users")
        cReq, _ := http.NewRequest("POST", createURL, bytes.NewReader(body))
        s.auth(cReq)
        cResp, err := s.httpClient.Do(cReq)
        if err != nil { return err }
        defer cResp.Body.Close()
        if cResp.StatusCode >= 300 { return fmt.Errorf("gitea create failed: %d", cResp.StatusCode) }
        return nil
    }

    return fmt.Errorf("gitea get user failed: %d", resp.StatusCode)
}

func (s *giteaServiceImpl) SyncAllUsers() (created, updated, skipped int, err error) {
    if !s.cfg.Gitea.Enabled { return 0,0,0, nil }
    // List users from DB in pages
    userService := NewUserService()
    page := 1
    pageSize := 100
    for {
        users, total, e := userService.GetUsers(page, pageSize)
        if e != nil { err = e; return }
        if len(users) == 0 { break }
        for _, u := range users {
            if e := s.EnsureUser(u); e != nil {
                logrus.WithError(e).WithField("username", u.Username).Warn("gitea ensure user failed")
                skipped++
            } else {
                // naive accounting: we can't distinguish created vs updated without extra GET
                updated++
            }
        }
        if int64(page*pageSize) >= total { break }
        page++
    }
    return
}

// simple random password (not critical; reverse proxy auth is used)
func randomPassword() string {
    return fmt.Sprintf("Temp!%d", time.Now().UnixNano())
}
