require "resty.core"
local redis = require "resty.redis"

-- この例ではリクエストURIから数字を抽出します
local uri = ngx.var.uri

local match, err = ngx.re.match(uri, "/api/chair/([0-9]+)", "jo")
--
--if not match then
--    if err then
--        ngx.log(ngx.ERR, "nginx lua 正規表現マッチングエラー: ", err)
--    end
--    ngx.exit(ngx.HTTP_NOT_FOUND)
--end

--local chair_id = string.match(uri, "/api/chair/(%d+)")

-- マッチした場合、マッチした値を取得します
local chair_id = match[1]

local red = redis:new()
red:set_timeout(1000)  -- 1秒のタイムアウト

local ok, err = red:connect("127.0.0.1", 6379)
if not ok then
    ngx.log(ngx.ERR, "Failed to connect to Redis: ", err)
    return
end

local res, err = red:sismember("sold_out_chair", chair_id)
if err then
    ngx.log(ngx.ERR, "Failed to check Redis: ", err)
    return
end

if res == 1 then
    ngx.log(ngx.INFO, "nginx lua で redisから売り切れの椅子を確認しました: " .. chair_id)
    red:set_keepalive(10000, 100) -- 接続をプールに戻す
    ngx.exit(ngx.HTTP_NOT_FOUND)
else
    -- アイテムが売り切れていない場合の処理をここに書く
end

-- 接続をプーリングに設定する
local ok, err = red:set_keepalive(10000, 100) -- 10秒のアイドルタイムアウト、
