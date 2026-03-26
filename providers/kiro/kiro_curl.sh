#!/bin/bash
#!/bin/bash

# 生成必要的 UUID
CONVERSATION_ID=$(uuidgen)
INVOCATION_ID=$(uuidgen)

# 设置核心参数
ACCESS_TOKEN="aoaAAAAAGmrLiwM5zGO-bG5IxwcoSwdoVrqq9VPpsbem_3lP-HMNe8ahJivwamPIS0qAmqY-chEHlmLiUK9buSeOMBkc0:MGUCMCepKZdZy5EXHKIkzuG74AWn+/FXCKM7unBXSjKtH6erjv8Yrpzz8zqLmzddNhyF9gIxAMj945x92EnmZs38yDIKeopdiaAM/L/GWPwNVmvTQSpBRr8CZugwV90hkLWfHdUWFw"
REGION="us-east-1"
BASE_URL="https://q.${REGION}.amazonaws.com/generateAssistantResponse"

# 执行 curl 请求（通过 @ 文件路径 读取 JSON 数据）
curl -X POST "${BASE_URL}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "amz-sdk-invocation-id: ${INVOCATION_ID}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -i \
  -d '{"conversationState":{"chatTriggerType":"MANUAL","conversationId":"f8c372f1-55d9-4b10-a661-f250410a207a","currentMessage":{"userInputMessage":{"content":"请介绍一下人工智能的基本概念和发展历程。","modelId":"claude-haiku-4.5","origin":"AI_EDITOR","userInputMessageContext":{"tools":[{"toolSpecification":{"name":"no_tool_available","description":"This is a placeholder tool when no other tools are available. It does nothing.","inputSchema":{"json":{"properties":{},"type":"object"}}}}]}}},"history":[{"userInputMessage":{"content":"你是一个有用的AI助手，请用中文回答用户的问题。\n\n请介绍一下人工智能的基本概念和发展历程。","modelId":"claude-haiku-4.5","origin":"AI_EDITOR"}},{"assistantResponseMessage":{"content":"Continue"}}]},"profileArn":"arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"}'

# 关键：用 @ 读取文件中的 JSON，避免转义问题

# # 设置请求参数
# ACCESS_TOKEN="aoaAAAAAGmrKNsitVBlP_bYkEt8Mq14uJqQ-zmC4G4VMaZI2-3chVHkVaZryswuQqaJj_0L6Gfs8mhhgIYcKJHwBkBkc0:MGUCMQC7vUc+D82jXnQjEBld1EEQSoaCgkdY/BbGCLuZTsRutpypu4HkkI5rkWD9IlRWYYkCMCRtHCc00L9xhr6sK5YmZXg0lbxZwJCJ6VNCj/FKuk3YJ7F+vV+ZmY6hkz0jwbtXrg"
# REGION="us-east-1"
# MODEL="claude-sonnet-4.5"
# CONVERSATION_ID=$(uuidgen)

# # 构建请求URL
# BASE_URL="https://q.${REGION}.amazonaws.com/generateAssistantResponse"

# # 构建请求头
# HEADERS=(
#     "-H" "Authorization: Bearer ${ACCESS_TOKEN}"
#     "-H" "amz-sdk-invocation-id: $(uuidgen)"
#     "-H" "Content-Type: application/json"
#     "-H" "Accept: application/json"
# )

# # 构建带工具规格的请求体
# REQUEST_BODY=$(cat << EOF
# {
#   "conversationState": {
#     "chatTriggerType": "MANUAL",
#     "conversationId": "${CONVERSATION_ID}",
#     "history": [
#       {
#         "userInputMessage": {
#           "content": "请帮我查询今天的天气信息",
#           "modelId": "CLAUDE_SONNET_4_5_20250929_V1_0",
#           "origin": "AI_EDITOR",
#           "userInputMessageContext": {
#             "tools": [
#               {
#                 "toolSpecification": {
#                   "name": "get_weather",
#                   "description": "查询指定城市的天气信息，包括温度、湿度、天气状况等",
#                   "inputSchema": {
#                     "json": {
#                       "type": "object",
#                       "properties": {
#                         "city": {
#                           "type": "string",
#                           "description": "城市名称"
#                         },
#                         "date": {
#                           "type": "string",
#                           "description": "日期，格式为YYYY-MM-DD"
#                         }
#                       },
#                       "required": ["city"]
#                     }
#                   }
#                 }
#               },
#               {
#                 "toolSpecification": {
#                   "name": "search_web",
#                   "description": "在互联网上搜索相关信息",
#                   "inputSchema": {
#                     "json": {
#                       "type": "object",
#                       "properties": {
#                         "query": {
#                           "type": "string",
#                           "description": "搜索关键词"
#                         }
#                       },
#                       "required": ["query"]
#                     }
#                   }
#                 }
#               }
#             ]
#           }
#         }
#       },
#       {
#         "assistantResponseMessage": {
#           "content": "我将使用天气查询工具来获取今天的天气信息。",
#           "toolUses": [
#             {
#               "toolUseId": "tool-001",
#               "name": "get_weather",
#               "input": "{\"city\": \"北京\", \"date\": \"2026-03-05\"}"
#             }
#           ]
#         }
#       }
#     ],
#     "currentMessage": {
#       "userInputMessage": {
#         "content": "请继续查询上海的天气",
#         "modelId": "CLAUDE_SONNET_4_5_20250929_V1_0",
#         "origin": "AI_EDITOR",
#         "userInputMessageContext": {
#           "toolResults": [
#             {
#               "toolUseId": "tool-001",
#               "content": [
#                 {
#                   "text": "{\"city\": \"北京\", \"date\": \"2026-03-05\", \"temperature\": \"15°C\", \"humidity\": \"45%\", \"condition\": \"晴\"}"
#                 }
#               ],
#               "status": "success"
#             }
#           ],
#           "tools": [
#             {
#               "toolSpecification": {
#                 "name": "get_weather",
#                 "description": "查询指定城市的天气信息，包括温度、湿度、天气状况等",
#                 "inputSchema": {
#                   "json": {
#                     "type": "object",
#                     "properties": {
#                       "city": {
#                         "type": "string",
#                         "description": "城市名称"
#                       },
#                       "date": {
#                         "type": "string",
#                         "description": "日期，格式为YYYY-MM-DD"
#                       }
#                     },
#                     "required": ["city"]
#                   }
#                 }
#               }
#             }
#           ]
#         }
#       }
#     }
#   }
# }
# EOF
# )

# # -- response
# # {
# #   "assistantResponse": {
# #     "content": "我将为您查询上海的天气信息。",
# #     "toolUses": [
# #       {
# #         "toolUseId": "tool-002",
# #         "name": "get_weather",
# #         "input": "{\"city\": \"上海\", \"date\": \"2026-03-05\"}"
# #       }
# #     ]
# #   },
# #   "conversationId": "${CONVERSATION_ID}",
# #   "usage": {
# #     "inputTokens": 312,
# #     "outputTokens": 45,
# #     "totalTokens": 357
# #   }
# # }
