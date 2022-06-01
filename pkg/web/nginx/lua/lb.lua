local _M = {}

local balancer = require "ngx.balancer"
local cache_storage = ngx.shared.dns_ip_cache_storage

function _M.set_domain(hostname, port)
      if not hostname or not port then
          return ngx.log(ngx.ERR, "Please enter a hostname or port")
      end

      local ip, _ = cache_storage:get(hostname)
      if ip then
         ngx.log(ngx.NOTICE, "set domain: ", hostname, " set IP: ", ip, " Port: ", port)
         _M.set_current_peer(ip, port)
      end
end

function _M.set_current_peer(ip, port)
      if not ip or not port then
          return ngx.log(ngx.ERR, "Please enter a ip address or port")
      end

      local ok, err = balancer.set_current_peer(ip, port)
      if not ok then
          ngx.log(ngx.ERR, "failed to set the current peer: ", err)
          return ngx.exit(ngx.ERROR)
      end
end

return _M
