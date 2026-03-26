-- token 格式
{
    "accessToken": "aoaAAAAAGmn9X0b0L69SEPINnvgceiOHrZPxQyG44R3vW-Y6-2Bk-CrVJKka8PV99pYiMZOMBUaK6l4Pd0X-hCugsBkc0:MGQCMBnmcpkuOSuz43tz6WL8VXbHPzcUcThzCxd1QN3ZYuFomNTwtGxLgg0Watc2lo/DfQIwX0mipEvExxpgCZirIccFXBEi7P3fSZ/Y+Q2wQ4cY/dZRhCRTtvu1mAEye8JRqzD7",
    "refreshToken": "aorAAAAAGnAj5EeTK3L-eX3QAIXQlcewYVa5FyoXS93W_ArsnPqeRgNyzdxJ1XG3Ds-T85UOPDCeILYGgHyCS1HjEBkc0:MGUCMQDCV6tqGbsz1rmXzkoASn2tspeEkbA6a5PTvE9zMytRwgSyq12B88++LNZhIzCI89wCMFY6M/wNDBFvjBg+61hgiEyxd/swJW5zTXl/70/LFZ/28R8p1VGo2VGAtLmZTJaiPA",
    "profileArn": "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK",
    "expiresAt": "2026-03-04T09:03:58.277426265Z",
    "authMethod": "social",
    "provider": "Google",
    "region": "us-east-1"
}

-- kiro refreshToken
curl -X POST \
  "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken" \
  -H "Content-Type: application/json" \
  -H "User-Agent: KiroIDE" \
  -d '{
    "refreshToken": "aorAAAAAGnAj5EeTK3L-eX3QAIXQlcewYVa5FyoXS93W_ArsnPqeRgNyzdxJ1XG3Ds-T85UOPDCeILYGgHyCS1HjEBkc0:MGUCMQDCV6tqGbsz1rmXzkoASn2tspeEkbA6a5PTvE9zMytRwgSyq12B88++LNZhIzCI89wCMFY6M/wNDBFvjBg+61hgiEyxd/swJW5zTXl/70/LFZ/28R8p1VGo2VGAtLmZTJaiPA"
  }'

  -- response
  {
    "accessToken":"aoaAAAAAGmpK8E-t1KQAFJJSF2WVhxy2CaOFChPvtgxrkJjzG40fx8PK4jr1u6jMg3lG4sN2VMYTBehWDERaIxA_gBkc0:MGYCMQCUSJpkX/2vKftKRJM1/LM7lD/z52VCmNZH/bIDXStGvc2/xS0aaH91Bf43FHbSoR4CMQCwIgEQw8YghcEX5XX0gBkrcdt0g+7pgzKHfhJs1mV89DH1tOoxKhxSgCNXXVpfvvA",
    "expiresIn":3600,
    "profileArn":"arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK","refreshToken":"aorAAAAAGnAj5EeTK3L-eX3QAIXQlcewYVa5FyoXS93W_ArsnPqeRgNyzdxJ1XG3Ds-T85UOPDCeILYGgHyCS1HjEBkc0:MGUCMQDCV6tqGbsz1rmXzkoASn2tspeEkbA6a5PTvE9zMytRwgSyq12B88++LNZhIzCI89wCMFY6M/wNDBFvjBg+61hgiEyxd/swJW5zTXl/70/LFZ/28R8p1VGo2VGAtLmZTJaiPA"
}

-- 401 Unauthorized
{
  "error": "unauthorized",
  "error_description": "Invalid credentials"
}

-- 400 Bad Request
{
  "error": "invalid_refresh_token",
  "error_description": "The refresh token is invalid or expired"
}



-- AWS SSO refreshToken
-- token格式
{
    "clientId": "123456",
    "clientSecret": "<KEY>",
    "accessToken": "<KEY>",
    "refreshToken": "<KEY>"
    "expiresAt": "2021-01-01T00:00:00.000Z"
}

-- request
curl -X POST \
  "https://oidc.us-east-1.amazonaws.com/token" \
  -H "Content-Type: application/json" \
  -H "User-Agent: KiroIDE" \
  -d '{
    "refreshToken": "aorAxxxxxxxxxxxxxxxxxxxxxxxx",
    "clientId": "client-id-from-aws-sso",
    "clientSecret": "client-secret-from-aws-sso",
    "grantType": "refresh_token"
  }'

  -- response
  {
  "accessToken": "aoaAAAAAGlfTyA8C4cV2FzIHN1Y2Nlc3NmdWxseSBnZW5lcmF0ZWQgdG9rZW4K",
  "refreshToken": "aorAxxxxxxxxxxxxxxxxxxxxxxxx",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "scope": "codewhisperer:completions codewhisperer:analysis"
}
