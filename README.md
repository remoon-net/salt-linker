# 简介

salt-linker 是服务于 [well-net](https://github.com/remoon-net/well) 的信令中继服务

# 自行部署

欢迎自行部署, 同时也可以用我部署的 [salt.remoon.cn](https://salt.remoon.cn) (是对项目一种支持)

## docker 运行

```sh
docker run --name salt-linker -p 8090:8090 shynome/salt-linker:v0.6.0
# 创建管理员用户. root@redacted-ip.invalid 是邮箱,  rootroot 是密码, 更换为你喜欢的值
docker exec -ti salt-linker /app/salt-linker superuser create root@redacted-ip.invalid rootroot
```

## 管理

打开后台管理页面 <http://127.0.0.1:8090/_/> 添加邮箱用户

然后回到 <http://127.0.0.1:8090/> 进行登录使用

## Caddy 反代

前端用户界面 [well.remoon.net](https://github.com/remoon-net/well.remoon.net), 你需要自行构建并放到对应的文件夹里

Caddyfile 示例文件:

```Caddyfile
well.remoon.net {
  root * /www/well.remoon.net/
  file_server
}

well.remoon.net/api/* {
  @uapi {
    # 禁止访问管理表
    not path /api/collections/_*
  }
  handle @uapi {
    reverse_proxy salt_linker:8090 {
      header_up X-Forwarded-For {http.request.header.X-Forwarded-For}
    }
  }
  respond 404
}

well-admin.remoon.net {
  # 使用自签客户端证书
	tls /etc/caddy/tls/self-sign.crt /etc/caddy/tls/self-sign.key {
	  client_auth {
	    trusted_ca_cert_file /etc/caddy/tls/staff.pem
	  }
	}
  handle {
    reverse_proxy salt_linker:8090
  }
}
```

注: well.remoon.net 并未部署过

# Todo

- [x] 计费充值系统

# 双重许可

如果你希望作品可以闭源发布, 可以向我购买闭源许可. (基于此贡献代码需要签署 CLA 允许我商业化售卖闭源许可)

当然如果作品是开源的, 只要遵守 AGPL3.0 许可即可, 将你的代码开放给软件使用者.
