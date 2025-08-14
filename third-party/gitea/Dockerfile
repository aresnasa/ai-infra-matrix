# Minimal wrapper to run Gitea with sensible defaults for the portal
# Multi-arch supported via upstream image
FROM gitea/gitea:1.22

# Copy custom entrypoint as root, then run as standard user
USER root
COPY gitea-entrypoint.sh /usr/local/bin/gitea-entrypoint.sh
RUN chmod +x /usr/local/bin/gitea-entrypoint.sh

# Use our wrapper entrypoint but delegate to upstream after init
ENTRYPOINT ["/usr/local/bin/gitea-entrypoint.sh"]

# No extra steps needed; configuration via env
