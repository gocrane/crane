local pcall = pcall
local io_open = io.open
local ngx_re_gmatch = ngx.re.gmatch

local ok, new_tab = pcall(require, "table.new")

if not ok then
    new_tab = function (narr, nrec) return {} end
end

ngx.shared.dns_servers = new_tab(5, 0)

local _read_file_data = function(path)
    local f, err = io_open(path, 'r')

    if not f or err then
        return nil, err
    end

    local data = f:read('*all')
    f:close()
    return data, nil
end

local _split = function(s, delimiter)
    result = {};
    for match in (s..delimiter):gmatch("(.-)"..delimiter) do
        table.insert(result, match);
    end
    return result;
end

local _read_dns_servers_from_resolv_file = function()
    local text = _read_file_data('/etc/resolv.conf')

    local captures, it, err
    it, err = ngx_re_gmatch(text, [[^nameserver\s+(\d+?\.\d+?\.\d+?\.\d+$)]], "jomi")

    for captures, err in it do
        if not err then
            ngx.shared.dns_servers[#ngx.shared.dns_servers + 1] = captures[1]
            ngx.log(ngx.NOTICE, "found dns server:", captures[1])
        end
    end

    local split_string = _split(text, " ")
    local cluster_domain = split_string[2]
    if cluster_domain then
        ngx.log(ngx.NOTICE, "cluster_domain:", cluster_domain)
        ngx.shared.cluster_domain = cluster_domain
    end
end

_read_dns_servers_from_resolv_file();
