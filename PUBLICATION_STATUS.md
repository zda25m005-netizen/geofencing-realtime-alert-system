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

- Frontend deployed on Render static hosting: https://boundary-dashboard-zda25m005.onrender.com
- Deployment source: private fork https://github.com/jhag8094/geofencing-realtime-alert-system (required because the Render GitHub App is installed on jhag8094).
- Browser verified live connection and saved fleet/geofence/rule/history data.
- Public API CORS matches the frontend origin. Two simultaneous WebSocket clients received entry and exit alerts; a repeated inside update produced no additional alert. PostgreSQL history persisted transitions.

## Remaining candidate steps

- Personal walkthrough uploaded: https://youtu.be/6_Qn5f0ib2c. Verified from a signed-out browser: Unlisted and accessible, but the player shows 3:08. This does not meet the required 30–45 minutes; replace it with the full recording before submission.
- All nine required reviewer invitations were sent and verified in GitHub. Access becomes active when each reviewer accepts.
- Submit the assessment form yourself with the verified links and refreshed source ZIP.

All hosting resources use free plans. Render PostgreSQL expires December 13, 2026. The free API can sleep when inactive and take time to wake up. The generated AI video remains optional reference material only.
