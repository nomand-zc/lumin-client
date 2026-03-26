-- request
curl -X POST \
  "https://auth.openai.com/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/json" \
  -H "User-Agent: CodexOAuth/1.0" \
  -d "grant_type=refresh_token&client_id=app_EMoamEEZ73f0CkXaXp7hrann&refresh_token=aorAxxxxxxxxxxxxxxxxxxxxxxxx"

  -- response
  {
  "access_token": "aoaAAAAAGlfTyA8C4cV2FzIHN1Y2Nlc3NmdWxseSBnZW5lcmF0ZWQgdG9rZW4K",
  "refresh_token": "aorAxxxxxxxxxxxxxxxxxxxxxxxx",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyXzEyMzQ1NiIsImVtYWlsIjoiZXhhbXBsZUBleGFtcGxlLmNvbSIsImh0dHBzOi8vYXBpLm9wZW5haS5jb20vYXV0aCI6eyJjaGF0Z3B0X2FjY291bnRfaWQiOiJhY2NvdW50XzEyMzQ1NiJ9LCJpYXQiOjE3MTIzNDU2NzgsImV4cCI6MTcxMjM0OTI3OH0",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid email profile offline_access"
}
