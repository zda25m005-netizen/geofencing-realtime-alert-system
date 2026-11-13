# Publication status

## Complete and verified

- Private GitHub repository: https://github.com/zda25m005-netizen/geofencing-realtime-alert-system
- GitHub Actions checks passed for Go, PostgreSQL/SQLite integration, React and Docker Compose.
- Both Docker images build and both containers start successfully.
- Data survives local container restart and recreation.
- PostgreSQL support and a free-only Render deployment blueprint are included.
- A 36 minute 30 second, explicitly AI-narrated walkthrough has been generated with actual app captures, source excerpts, chapters, subtitles and a transcript. It is supplied as a separate MP4 and private release asset.

- Published Docker Hub image: https://hub.docker.com/r/jhag8094/geofence-api — version `1.0.0`, Linux AMD64 and ARM64. Registry metadata and both platform manifests were verified.
- `jhag8094` accepted repository access and has write permission.

- Backend deployed on Render Free: https://boundary-api-zda25m005.onrender.com — GET /health verified successfully.
- Render PostgreSQL 18 database is Available in Singapore on the Free plan. It expires December 13, 2026.

## Not yet complete

- Frontend deployment: waiting for action-time approval of the GitHub Render authorization screen to connect the source repository.
- End-to-end deployed frontend and WebSocket checks remain pending.
- Candidate will record and upload their own 30–45 minute walkthrough; generated AI video is optional reference material only.
- Reviewer access invitations and final form submission have not been performed.

## Continue

1. Authorize Render's GitHub connection, restricting repository access to this project where possible.
2. Deploy the React dashboard as a free static site and configure VITE_API_URL with the verified backend URL.
3. Verify the frontend origin matches backend CORS_ORIGIN and check live entry/exit alerts.
4. Record and upload the candidate's video, refresh the submission ZIP, and submit verified links.

Do not submit a guessed service URL or the assignment's public repository URL as the solution.
