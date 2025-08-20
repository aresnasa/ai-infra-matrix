// Auto-redirect script for Gitea SSO integration
// This script automatically redirects authenticated users to their intended destination

(function() {
    'use strict';
    
    // Check if we're on the login page and user is already authenticated
    if (window.location.pathname === '/user/login' || window.location.pathname === '/gitea/user/login') {
        // Look for user avatar or any sign that user is logged in
        const userAvatar = document.querySelector('.ui.avatar, .navbar-avatar, [data-tooltip*="admin"], .user.avatar');
        const userMenu = document.querySelector('.ui.dropdown.user, .navbar-user-menu');
        const isLoggedIn = userAvatar || userMenu || document.querySelector('body').innerHTML.includes('admin@');
        
        if (isLoggedIn) {
            // Get redirect_to parameter from URL
            const urlParams = new URLSearchParams(window.location.search);
            const redirectTo = urlParams.get('redirect_to') || '/gitea/';
            
            console.log('SSO login detected, redirecting to:', redirectTo);
            
            // Redirect after a short delay to allow Gitea to finish initialization
            setTimeout(function() {
                window.location.href = redirectTo;
            }, 500);
        }
    }
})();
