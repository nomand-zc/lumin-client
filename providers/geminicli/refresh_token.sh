-- request
curl -X POST "https://oauth2.googleapis.com/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=your_client_id_here" \
  -d "client_secret=your_client_secret_here" \
  -d "refresh_token=your_refresh_token_here" \
  -d "grant_type=refresh_token"

  -- response
{
  "access_token": "ya29.a0AfH6SMB...",
  "expires_in": 3599,
  "scope": "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile",
  "token_type": "Bearer"
}
