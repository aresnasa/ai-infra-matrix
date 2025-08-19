COOKIE_JAR="/tmp/ai_infra_cookies_$$.txt"; \
LOGIN_PAYLOAD='{"username":"admin","password":"admin123"}'; \
# login to backend to get SSO cookies
curl -s -X POST -H 'Content-Type: application/json' -d "$LOGIN_PAYLOAD" -c "$COOKIE_JAR" http://localhost:8080/api/auth/login -o /dev/null -w 'login_status=%{http_code}\n'; \
# hit gitea login endpoint to trigger header-based login
curl -s -I -b "$COOKIE_JAR" -c "$COOKIE_JAR" http://localhost:8080/gitea/user/login | sed -n '1,20p'; \
# now try admin page (no follow)
curl -s -I -b "$COOKIE_JAR" -c "$COOKIE_JAR" http://localhost:8080/gitea/admin | sed -n '1,40p'; \
# try with follow redirects to see final status
curl -s -L -o /dev/null -w 'final_status=%{http_code} final_url=%{url_effective}\n' -b "$COOKIE_JAR" -c "$COOKIE_JAR" http://localhost:8080/gitea/admin; \
# dump any gitea cookies
printf '\nCookies saved:\n'; cat "$COOKIE_JAR" | sed -n '1,200p'; \
# show a couple of debug headers by fetching HTML head from /gitea/user/login
curl -s -D - -o /dev/null -b "$COOKIE_JAR" -c "$COOKIE_JAR" http://localhost:8080/gitea/user/login | grep -i '^x-debug' || true; \
# cleanup var path echo
printf "\nCOOKIE_JAR=$COOKIE_JAR\n"