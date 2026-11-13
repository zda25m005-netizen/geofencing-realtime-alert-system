# Publication and hosting

## GitHub

Solution repository: https://github.com/zda25m005-netizen/geofencing-realtime-alert-system

This repository must remain private. The included GitHub Actions workflow checks Go, React and the Docker Compose stack on push. Reviewer invitations are separate from creating the repository; verify recipients against the assignment brief before inviting them.

## Docker Hub

Sign in with your own Docker Hub account. Do not put credentials in files or chat.

```sh
docker login
sh scripts/publish-image.sh YOUR_DOCKERHUB_USERNAME 1.0.0
```

The script builds both Linux AMD64 and ARM64 variants and pushes the versioned backend image. Publication requires an authenticated account with write access to the chosen namespace.

## Render

The supplied `render.yaml` describes a Docker backend with a persistent disk and a static React frontend. It is prepared configuration, not evidence of deployment. **The backend uses a paid Starter service plus a 1 GB disk. Do not launch it without approving the current cost in your account.** Persistent disks are unavailable on free Render web services: https://render.com/docs/disks

1. Connect your Render account and grant it access only to this private solution repository.
2. Create a Blueprint from the repository after approving the service and disk cost.
3. Set the backend's `CORS_ORIGIN` to the exact frontend HTTPS origin, without trailing slash.
4. Set the frontend's `VITE_API_URL` to the actual backend HTTPS base URL. Rebuild the frontend after changing it.
5. Confirm `/health` responds and use two browser tabs to verify entry and exit delivery.
6. Restart the backend and verify history persists on its mounted disk.
7. Put the verified URLs in README.md and SUBMISSION.md. Do not use estimated service URLs.

The static frontend may be hosted elsewhere if desired; ensure its API URL and the backend CORS setting agree. For free-only hosting, choose a provider with durable storage or change the database implementation; an ephemeral SQLite deployment is not a durable solution.

## Video

`VIDEO_GUIDE.md` provides the required 30–45 minute outline. A genuine personal walkthrough still needs to be recorded, or an AI-narrated walkthrough must be clearly disclosed. Do not represent synthetic narration as the candidate's own explanation. Upload the final video as unlisted to an account you control, verify access and add its real URL.
