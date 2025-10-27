#!/bin/sh

regenerate_index() {
    echo "Regenerating package indexes..."
    
    # Regenerate deb index if packages exist
    if [ -d /usr/share/nginx/html/deb ]; then
        cd /usr/share/nginx/html/deb
        if ls *.deb >/dev/null 2>&1; then
            echo "Regenerating deb index..."
            dpkg-scanpackages . /dev/null > Packages
            gzip -f Packages
            echo "deb index regenerated"
        fi
    fi
    
    # Regenerate SLURM deb index
    if [ -d /usr/share/nginx/html/pkgs/slurm-deb ]; then
        cd /usr/share/nginx/html/pkgs/slurm-deb
        if ls *.deb >/dev/null 2>&1; then
            echo "Regenerating SLURM deb index..."
            dpkg-scanpackages . /dev/null > Packages
            gzip -c Packages > Packages.gz
            echo "SLURM deb index regenerated"
        fi
    fi

    # List rpm packages if they exist (no metadata generation in Alpine)
    if [ -d /usr/share/nginx/html/rpm ]; then
        cd /usr/share/nginx/html/rpm
        if ls *.rpm >/dev/null 2>&1; then
            echo "Found rpm packages (direct download available):"
            ls -lh *.rpm
            echo "Note: RPM metadata not generated (createrepo not available in Alpine)"
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