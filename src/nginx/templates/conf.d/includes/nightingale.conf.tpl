# Nightingale Monitoring System Proxy Configuration
# Nightingale backend is configured with BasePath="/nightingale"
# Frontend is built with VITE_PREFIX=/nightingale
# Template variables: NIGHTINGALE_HOST, NIGHTINGALE_PORT
# Note: /api/n9e/ is handled in server-main.conf to ensure proper priority

## IMPORTANT: Do NOT globally intercept top-level /js, /image, /font
## Doing so breaks the frontend SPA and can cause redirect loops.
## Nightingale assets will be served via the /nightingale/ location below.

# Redirect Nightingale assets from root to /nightingale/
# These are for assets that Nightingale hardcodes without the prefix
location = /js/widget.js {
    return 302 /nightingale/js/widget.js;
}

location ^~ /image/ {
    # Proxy to Nightingale for image assets
    rewrite ^/image/(.*)$ /nightingale/image/$1 last;
}

location ^~ /font/ {
    # Proxy to Nightingale for font assets
    rewrite ^/font/(.*)$ /nightingale/font/$1 last;
}

# Main Nightingale location - must come before regex locations
# Use ^~ to stop regex matching (prevents static file location from intercepting)
location ^~ /nightingale/ {
    # Proxy to Nightingale backend - NO trailing slash to preserve /nightingale prefix
    # Backend is configured with BasePath="/nightingale" to handle this
    proxy_pass http://{{NIGHTINGALE_HOST}}:{{NIGHTINGALE_PORT}};
    
    # Cookies should stay under /nightingale/
    proxy_cookie_path / /nightingale/;
    proxy_cookie_domain {{NIGHTINGALE_HOST}} $http_host;
    
    # SSO Integration: Extract username from JWT token via auth_request
    # The backend's /auth/verify endpoint returns X-User header with username
    # This enables ProxyAuth SSO - users are automatically logged into Nightingale
    auth_request /__auth/verify;
    auth_request_set $auth_username $upstream_http_x_user;
    
    # Pass the authenticated username to Nightingale for ProxyAuth
    proxy_set_header X-User-Name $auth_username;
    
    # Standard proxy headers
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $http_host;
    proxy_set_header X-Forwarded-Prefix /nightingale;
    
    # Hide iframe blocking headers
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    
    # Inject language sync script for iframe integration
    # This script reads lang parameter from URL and syncs it to localStorage
    # Nightingale uses localStorage key 'language' to store language preference
    sub_filter_once off;
    sub_filter_types text/html;
    sub_filter '</head>' '<script>
(function(){
  try {
    var urlParams = new URLSearchParams(window.location.search);
    var lang = urlParams.get("lang");
    console.log("[N9E-LangSync] URL lang param:", lang);
    
    if (lang) {
      // Nightingale 使用 en_US / zh_CN 格式
      var n9eLang = (lang === "en" || lang === "en-US" || lang === "en_US") ? "en_US" : "zh_CN";
      var currentLang = localStorage.getItem("language");
      console.log("[N9E-LangSync] Current localStorage language:", currentLang, "Target:", n9eLang);
      
      // 强制设置语言，无论当前是什么
      if (currentLang !== n9eLang) {
        console.log("[N9E-LangSync] Setting language to:", n9eLang);
        localStorage.setItem("language", n9eLang);
        
        // 标记需要重新加载（避免无限循环）
        var reloadFlag = sessionStorage.getItem("n9e_lang_reload");
        var reloadKey = "reload_" + n9eLang;
        
        if (reloadFlag !== reloadKey) {
          sessionStorage.setItem("n9e_lang_reload", reloadKey);
          console.log("[N9E-LangSync] Language changed, reloading page...");
          setTimeout(function() {
            window.location.reload();
          }, 100);
        } else {
          console.log("[N9E-LangSync] Already reloaded for this language, skipping");
        }
      } else {
        console.log("[N9E-LangSync] Language already correct:", n9eLang);
        // 清除重新加载标记
        sessionStorage.removeItem("n9e_lang_reload");
      }
    }
    var theme = urlParams.get("themeMode");
    if (theme) {
      var n9eTheme = (theme === "dark") ? "dark" : "light";
      var currentTheme = localStorage.getItem("theme");
      if (currentTheme !== n9eTheme) {
        console.log("[N9E-LangSync] Setting theme to:", n9eTheme);
        localStorage.setItem("theme", n9eTheme);
      }
    }
  } catch(e) { console.error("[N9E-LangSync] Error:", e); }
})();
</script></head>';
    
    # WebSocket support
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # Disable response buffering for sub_filter to work
    proxy_buffering on;
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;
    
    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
}

