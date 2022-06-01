local _M = {}
local require = require
local ngx_re_find = ngx.re.find
local lrucache = require "resty.lrucache"
local resolver = require "resty.dns.resolver"
local cache_storage = ngx.shared.dns_ip_cache_storage

function _M.is_addr(hostname)
    return ngx_re_find(hostname, [[\d+?\.\d+?\.\d+?\.\d+$]], "jo")
end

function _M.get_addr(hostname)
    if _M.is_addr(hostname) then
        return hostname, hostname
    end

    local addr, _ = cache_storage:get(hostname)

    if addr then
        return addr, hostname
    end

    local r, err = resolver:new({
        nameservers = ngx.shared.dns_servers,
        retrans = 5,  -- 5 retransmissions on receive timeout
        timeout = 2000,  -- 2 sec
    })

    if not r then
        return nil, hostname
    end

    local query_string = string.format("%s.%s",hostname, ngx.shared.cluster_domain)

    local answers, err = r:query(query_string, {qtype = r.TYPE_A})

    if not answers or answers.errcode then
        ngx.log(ngx.ERR,"Failed to query hostname: ", query_string, " errcode: ", answers.errcode)
        return nil, hostname
    end

    for i, ans in ipairs(answers) do
        if ans.address then
            cache_storage:set(hostname, ans.address, 300)
            ngx.log(ngx.NOTICE,"Query DNS: ", hostname , " => ", ans.address)
            return ans.address, hostname
        end
    end

    return nil, hostname
end

return _M
