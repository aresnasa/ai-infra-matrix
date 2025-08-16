// Utility to resolve which service the SSO flow should target (jupyter or gitea)
// Based on current pathname, query params, or explicit hints in next/redirect_uri

const detectFromString = (str) => {
  if (!str || typeof str !== 'string') return null;
  const s = str.toLowerCase();
  if (s.includes('/gitea')) return 'gitea';
  if (s.includes('/jupyter') || s.includes('jupyterhub')) return 'jupyter';
  return null;
};

export function resolveSSOTarget() {
  try {
    const { pathname, search } = window.location;
    const params = new URLSearchParams(search);

    // Priority 1: explicit query hints
    const qp = params.get('target') || params.get('service') || params.get('next') || params.get('redirect_uri');
    const hinted = detectFromString(qp);
    if (hinted) return buildTarget(hinted);

    // Priority 2: current path
    const byPath = detectFromString(pathname);
    if (byPath) return buildTarget(byPath);

    // Default to jupyter
    return buildTarget('jupyter');
  } catch (_) {
    return buildTarget('jupyter');
  }
}

function buildTarget(kind) {
  if (kind === 'gitea') {
    return {
      key: 'gitea',
      name: 'Gitea',
      // Where SSO should bring the user after auth
      nextPath: '/gitea/',
      // For some flows that used jupyterhub-authenticated previously
      authenticatedPath: '/gitea/',
    };
  }
  // default jupyter
  return {
    key: 'jupyter',
    name: 'JupyterHub',
    nextPath: '/jupyter/hub/',
    authenticatedPath: '/jupyterhub-authenticated',
  };
}

export default resolveSSOTarget;
