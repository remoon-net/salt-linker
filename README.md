# 自行部署

```sh
docker run --name salt-linker -p 8090:8090 shynome/salt-linker:v0.2.0
# 创建管理员用户. root@redacted-ip.invalid 是邮箱,  rootroot 是密码, 更换为你喜欢的值
docker exec -ti salt-linker /app/salt-linker superuser create root@redacted-ip.invalid rootroot
```

打开后台管理页面 <http://127.0.0.1:8090/_/> 添加邮箱用户

然后回到 <http://127.0.0.1:8090/> 进行登录使用 (UI 界面未开源)

# Todo

- [ ] 计费充值系统
