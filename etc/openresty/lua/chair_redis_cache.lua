local redis = require "resty.redis"
local red = redis:new()

local function error_handler(err)
    ngx.log(ngx.ERR, "nginx lua でエラー発生: ", err)
    -- 必要に応じて、HTTPレスポンスをカスタマイズする
    ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
    ngx.say("nginx lua で サーバーエラーが発生しました。")
    ngx.exit(ngx.status)
end


red:set_timeout(1000)  -- 1秒のタイムアウト

local ok, err = xpcall(red:connect("127.0.0.1", 6379), error_handler)
if not ok then
    ngx.say("Failed to connect to Redis: ", err)
    return
end

local res, err = red:get("your_key")
if not res then
    return
else
    ngx.log(ngx.INFO,"nginx lua で redisから売り切れの椅子を取得しました: " .. res)
    ngx.status = HTTP_NOT_FOUND
    ngx.exit(ngx.HTTP_NOT_FOUND)
end

-- 接続をプーリングに設定する
local ok, err = xpcall(red:set_keepalive(10000, 100), error_handler) -- 10秒のアイドルタイムアウト、100個の接続をプール
if not ok then
    ngx.say("Failed to set keepalive: ", err)
    return
end