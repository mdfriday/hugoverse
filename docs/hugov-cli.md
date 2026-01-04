# Hugoverse Setup

[]: Check Hugoverse PID: lsof -i :1314
[]: Start serve in Prod: nohup go/bin/hugoverse serve -env prod &
[]: Config CouchDB: http://localhost:5984 , admin , admin password, userdb- prefix
[]: Config Caddy: mdfriday.sunwei.xyz, 80 
[]: Save and Restart Hugoverse
[]: go/bin/hugoverse caddy start -domain mdfriday.sunwei.xyz
[]: go/bin/hugoverse caddy add -domain sunwei.xyz -path /web/sunwei-xyz-raw
[]: go/bin/hugoverse caddy add -domain notes.sunwei.xyz -path /web/sunwei-xyz-raw/notes
[]: go/bin/hugoverse license generate -email email@sunwei.xyz -password pwd -plan starter -count 5


```shell
sunwei.xyz {
	root * /web/sunwei-xyz-raw
	file_server
}

notes.sunwei.xyz {
	root * /web/sunwei-xyz-raw/notes
	file_server
}

# 子域名配置，将所有请求转发到 127.0.0.1:1314
mdfriday.sunwei.xyz {
    reverse_proxy 127.0.0.1:1314
}
```


## TODO

### 解决 SSL 证书获取失败的问题
```shell
root@iZwz908qk7jina1lucvusaZ:~# go/bin/hugoverse caddy cert -domain sunwei.xyz
📜 Checking SSL certificate for: sunwei.xyz

Error: failed to get certificate status: failed to get certificates (status 400): {"error":"invalid traversal path at: config/apps/tls/certificates"}
```

### 子域名解析需要先配置 DNS ，否则访问不了
### 子域名证书不是 *.sunwei.xyz 形式的泛域名证书，而是单独申请的证书，可能会有数量限制