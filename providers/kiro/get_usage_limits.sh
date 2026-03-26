-- request
curl -X GET "https://q.us-east-1.amazonaws.com/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&resourceType=AGENTIC_REQUEST&profileArn=arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK" \
  -H "Authorization: Bearer aoaAAAAAGmq7F4q3U_Ciy0IH1jeFh_mQGSPyit38EknMDaEeJgCaxvWa3FwFHNhkzHFOo5UZ5_UwZvdGdBzY3u1NEBkc0:MGUCMQCJ5tBJwOTzz3MW6BVpgXGMK5rnWyKbBebuu3faU/NC+SbfmEl9eyOYBAQlQMrHqp8CMCTtYnGOrnevfLZHq3hP682wFRjBiLwUS7qdQ6lk8PjYVusDqVIV3lNjdJetKZhr1A" \
  -H "x-amz-user-agent: aws-sdk-js/1.0.0 KiroIDE-0.8.140-machine123456" \
  -H "user-agent: aws-sdk-js/1.0.0 ua/2.1 os/Linux lang/js md/nodejs#18.17.0 api/codewhispererruntime#1.0.0 m/E KiroIDE-0.8.140-machine123456" \
  -H "amz-sdk-invocation-id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "amz-sdk-request: attempt=1; max=1" \
  -H "Connection: close"

-- response
{
	"daysUntilReset": 0, // 距离使用限制重置还有多少天（0表示今天重置）
	"limits": [], // 限制列表（当前为空）

	"nextDateReset": 1.7750016E9, // 下次重置的时间戳（Unix时间戳格式）

  // 超额配置
  "overageConfiguration": {
    "overageLimit": null,           // 超额限制（null表示未设置）
    "overageStatus": "DISABLED"     // 超额状态（禁用）
  },

  // 订阅信息
	"subscriptionInfo": {
    "overageCapability": "OVERAGE_INCAPABLE",    // 是否支持超额使用（不支持）
    "subscriptionManagementTarget": "PURCHASE",  // 订阅管理目标（购买）
    "subscriptionTitle": "KIRO FREE",           // 订阅标题（免费版）
    "type": "Q_DEVELOPER_STANDALONE_FREE",      // 订阅类型（开发者独立免费版）
    "upgradeCapability": "UPGRADE_CAPABLE"      // 是否支持升级（支持）
  },

	"totalUsage": null, // 总使用量（null表示未提供）
	"usageBreakdown": null, // 使用细分（null表示未提供）

  // 使用限制详情
	"usageBreakdownList": [{
    "bonuses": [],                    // 奖励额度，当前为空数组
    "currency": "USD",               // 计费货币单位
    "currentOverages": 0,            // 当前超额使用量
    "currentOveragesWithPrecision": 0.0, // 精确的超额使用量
    "currentUsage": 20,              // 当前使用量（整数）
    "currentUsageWithPrecision": 20.13, // 精确的当前使用量
    "displayName": "Credit",         // 显示名称（单数）
    "displayNamePlural": "Credits",  // 显示名称（复数）
    "freeTrialInfo": {               // 免费试用信息
      "currentUsage": 378,           // 试用期间使用量
      "currentUsageWithPrecision": 378.93, // 精确的试用使用量
      "freeTrialExpiry": 1.771667182617E9, // 试用到期时间戳
      "freeTrialStatus": "EXPIRED",  // 试用状态（已过期）
      "usageLimit": 500,             // 试用期间总限额
      "usageLimitWithPrecision": 500.0 // 精确的试用限额
    },
    "overageCap": 10000,             // 超额使用上限
    "overageCapWithPrecision": 10000.0, // 精确的超额上限
    "overageCharges": 0.0,           // 超额费用
    "overageRate": 0.04,             // 超额费率（每单位0.04美元）
    "resourceType": "CREDIT",        // 资源类型（信用额度）
    "unit": "INVOCATIONS",           // 计量单位（调用次数）
    "usageLimit": 50,                // 正常使用限额
    "usageLimitWithPrecision": 50.0  // 精确的使用限额
  }],
	"userInfo": {
    "email": "ditaprasetyo881@gmail.com",                    // 用户邮箱
    "userId": "d-9067c98495.14c81498-d0a1-704b-db86-298a486ea0b4" // 用户ID
  }
}

-- 401 Unauthorized
{
  "message":"The bearer token included in the request is invalid.",
  "reason":null
}

-- 403 Forbidden
{
  "error": "Forbidden",
  "message": "Account temporarily suspended"
}

-- 429 Too Many Requests
{
  "error": "RateLimitExceeded",
  "message": "Too many requests, please try again later",
  "retryAfter": 60
}
