FROM nginx:stable

# Install tools for repo management (deb packages only)  
RUN apt-get update && \
    apt-get install -y apt-utils dpkg-dev && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

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