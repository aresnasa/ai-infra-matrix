# SaltStack NetAPI Troubleshooting

## Symptom: 404 Not Found — The path '/app' was not found (CherryPy)

- Cause: rest_cherrypy was configured with `app`/`app_path: /app`, which mounts a static app and changes how the root path (`/`) behaves. NetAPI POSTs to `/` then may 404 with messages mentioning `/app`.

### Fixes applied

- Backend hardening:
  - Strip any path segment from `SALTSTACK_MASTER_URL` so only scheme://host:port is used.
  - Retry NetAPI POSTs to `/run` when POSTing to `/` returns 404.
- Salt API config:
  - Disable or change static app mount in `src/saltstack/salt-api.conf` (commented out `app` and `app_path`).

### Validate

- `curl -s http://saltstack:8002/` returns JSON.
- `curl -s -X POST http://saltstack:8002/ -H 'Content-Type: application/json' -d '{"client":"runner","fun":"manage.status"}'` returns 200 with `{"return":...}`.
- From the portal: `/api/saltstack/status` returns 200 with data.

### Notes

- If you must keep a static app, prefer a distinct mount path (e.g. `/saltapp`) and leave root for NetAPI.
