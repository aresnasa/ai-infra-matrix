FROM nginx:alpine

# Install tools for repo management (deb packages only)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache dpkg-dev

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Create directories for packages
RUN mkdir -p /usr/share/nginx/html/deb /usr/share/nginx/html/rpm

# Expose port
EXPOSE 80

# Entrypoint to regenerate indexes if needed
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]