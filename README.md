# MY-ZINX

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Meha555/my-zinx)

TCP长连接框架，包含服务器部分和客户端部分。

特点：
- 可选的外部依赖（uuid\gid）
- 支持心跳检测
- 支持自定义协议
- 支持自定义消息
- 支持自定义路由
- 支持自定义连接

## 主要模块

- server 服务器接口
- client 客户端接口
- router 路由接口
- message 消息接口（不含协议）
- job/task 任务接口
- workerpool 协程池接口
- session+connection 连接管理接口
- log 日志
- config 配置
