#!/bin/sh

regenerate_index() {
    echo "Regenerating package indexes..."
    
    # Regenerate deb index if packages exist
    if [ -d /usr/share/nginx/html/deb ]; then
        cd /usr/share/nginx/html/deb
        if ls *.deb >/dev/null 2>&1; then
            echo "Regenerating deb index..."
            apt-ftparchive packages . > Packages
            gzip -f Packages
            echo "deb index regenerated"
        fi
    fi

    # Regenerate rpm index if packages exist
    if [ -d /usr/share/nginx/html/rpm ]; then
        cd /usr/share/nginx/html/rpm
        if ls *.rpm >/dev/null 2>&1; then
            echo "Regenerating rpm index..."
            createrepo .
            echo "rpm index regenerated"
        fi
    fi
}

# Check if called with regenerate-index argument
if [ "$1" = "regenerate-index" ]; then
    regenerate_index
    exit 0
fi

# Normal startup - regenerate indexes first
regenerate_index

# Start nginx
nginx -g 'daemon off;'