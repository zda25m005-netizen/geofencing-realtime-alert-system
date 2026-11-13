# Publication and hosting

## GitHub

Solution repository: https://github.com/zda25m005-netizen/geofencing-realtime-alert-system

This repository must remain private. The included GitHub Actions workflow checks Go, React and the Docker Compose stack on push. Reviewer invitations are separate from creating the repository; verify recipients against the assignment brief before inviting them.

## Docker Hub

Published: https://hub.docker.com/r/jhag8094/geofence-api — tag `1.0.0`, supporting Linux AMD64 and ARM64. The registry manifests were verified. The following commands are for publishing a future version.

Sign in with your own Docker Hub account. Do not put credentials in files or chat.

```sh
docker login
sh scripts/publish-image.sh YOUR_DOCKERHUB_USERNAME 1.0.0
```

The script builds both Linux AMD64 and ARM64 variants and pushes the versioned backend image. Publication requires an authenticated account with write access to the chosen namespace.

## Render

The supplied `render.yaml` uses only free plans: a Docker API, a separate PostgreSQL database and a static frontend. No paid services or disks are configured. The database URL is injected by Render; no database credential is committed.

1. Connect your Render account and grant it access only to this private solution repository.
2. Create a Blueprint from the repository, choosing the free resources in `render.yaml`.
3. Set the backend's `CORS_ORIGIN` to the exact frontend HTTPS origin, without trailing slash.
4. Set the frontend's `VITE_API_URL` to the actual backend HTTPS base URL. Rebuild after changing it.
5. Confirm `/health` responds and use two browser tabs to verify entry and exit delivery.
6. Restart the backend and verify history persists in PostgreSQL.
7. Put the verified URLs in README.md and SUBMISSION.md. Do not use estimated service URLs.

Free Render PostgreSQL expires 30 days after creation, and free web services sleep after inactivity. This is time-limited assessment hosting, not permanent production hosting. Do not upgrade to a paid plan automatically. Official limitations: https://render.com/docs/free

A pre-existing free PostgreSQL provider can also be used by setting its `DATABASE_URL`. Never deploy SQLite to an ephemeral filesystem and claim its data survives restarts.

## Video

`VIDEO_GUIDE.md` provides the required 30–45 minute outline. A 36:30 AI-narrated walkthrough is generated and clearly disclosed; see the private release assets. Do not represent synthetic narration as the candidate's own explanation. Upload the final video as unlisted to an account you control, verify access and add its real URL.

## Verified deployment

Frontend: https://boundary-dashboard-zda25m005.onrender.com
Backend: https://boundary-api-zda25m005.onrender.com
Health check: https://boundary-api-zda25m005.onrender.com/health

Frontend is built from the private `jhag8094/geofencing-realtime-alert-system` fork, root `frontend`, build `npm ci && npm run build`, publish `dist`, Node 22, VITE_API_URL set to the backend URL. Backend uses `jhag8094/geofence-api:1.0.0` with PORT=8080, CORS_ORIGIN set to the frontend origin, and Render's internal PostgreSQL connection stored only in the Render environment. Both resources use free plans; PostgreSQL expires December 13, 2026.
