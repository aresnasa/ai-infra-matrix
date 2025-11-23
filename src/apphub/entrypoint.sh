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

    # List rpm packages if they exist
    if [ -d /usr/share/nginx/html/rpm ]; then
        cd /usr/share/nginx/html/rpm
        if ls *.rpm >/dev/null 2>&1; then
            echo "Found rpm packages:"
            ls -lh *.rpm
            
            # Try to generate RPM metadata if createrepo is available
            if command -v createrepo >/dev/null 2>&1; then
                echo "Regenerating RPM metadata..."
                createrepo .
                echo "RPM metadata regenerated"
            elif command -v createrepo_c >/dev/null 2>&1; then
                echo "Regenerating RPM metadata (using createrepo_c)..."
                createrepo_c .
                echo "RPM metadata regenerated"
            else
                echo "Note: RPM metadata not generated (createrepo not installed)"
            fi
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

# Start SSH server (for backend to copy scripts)
echo "Starting SSH server..."
/usr/sbin/sshd
echo "✓ SSH server started on port 22"

# Start nginx
nginx -g 'daemon off;'