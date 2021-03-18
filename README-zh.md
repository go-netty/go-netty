# GO-NETTY

[![GoDoc][1]][2] [![license-Apache 2][3]][4] [![Go Report Card][5]][6] [![Build Status][9]][10] [![Coverage Status][11]][12]

<!--[![Downloads][7]][8]-->

[1]: https://godoc.org/github.com/go-netty/go-netty?status.svg
[2]: https://godoc.org/github.com/go-netty/go-netty
[3]: https://img.shields.io/badge/license-Apache%202-blue.svg
[4]: LICENSE
[5]: https://goreportcard.com/badge/github.com/go-netty/go-netty
[6]: https://goreportcard.com/report/github.com/go-netty/go-netty
[7]: https://img.shields.io/github/downloads/go-netty/go-netty/total.svg?maxAge=1800
[8]: https://github.com/go-netty/go-netty/releases
[9]: https://travis-ci.org/go-netty/go-netty.svg?branch=master
[10]: https://travis-ci.org/go-netty/go-netty
[11]: https://codecov.io/gh/go-netty/go-netty/branch/master/graph/badge.svg
[12]: https://codecov.io/gh/go-netty/go-netty

## 介绍

go-netty 是一款受netty启发的Go语言可扩展的高性能网络库

## 特性

* 可扩展多种传输协议，并且默认实现了 TCP, [UDP, QUIC, KCP, Websocket](https://github.com/go-netty/go-netty-transport)
* 可扩展多种解码器，默认实现了常见的编解码器
* 基于责任链模型的流程控制
* 核心库零依赖

## 文档
* [GoDoc](https://godoc.org/github.com/go-netty/go-netty)

## 示例

* [chat_server (基于websocket的聊天室)](https://github.com/go-netty/go-netty-samples/blob/master/chat_server/main.go)  
* [file_server (基于http的文件浏览器)](https://github.com/go-netty/go-netty-samples/blob/master/file_server/main.go)  
* [tcp_server (自义定tcp服务器)](https://github.com/go-netty/go-netty-samples/blob/master/tcp_server/main.go)  
* [redis_cli (简单的redis客户端)](https://github.com/go-netty/go-netty-samples/blob/master/redis_cli/main.go)
* [go-netty-samples (更多例子)](https://github.com/go-netty/go-netty-samples)  

## 快速开始
```go
package main

import (
	"fmt"
	"strings"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/format"
	"github.com/go-netty/go-netty/codec/frame"
)

func main() {

    // 子连接的流水线配置
    var childInitializer = func(channel netty.Channel) {
        channel.Pipeline().
            // 最大允许包长128字节，使用\n分割包, 丢弃分隔符
            AddLast(frame.DelimiterCodec(128, "\n", true)).
            // 解包出来的bytes转换为字符串
            AddLast(format.TextCodec()).
            // 日志处理器, 打印连接建立断开消息，收到的消息
            AddLast(LoggerHandler{}).
            // 业务处理器 (将字符串全部大写)
            AddLast(UpperHandler{})
    }

	// 创建Bootstrap & 监听端口 & 接受连接
	netty.NewBootstrap(netty.WithChildInitializer(childInitializer)).
		Listen(":9527").Sync()
}

type LoggerHandler struct {}

func (LoggerHandler) HandleActive(ctx netty.ActiveContext) {
    fmt.Println("go-netty:", "->", "active:", ctx.Channel().RemoteAddr())
    // 写入欢迎信息
    ctx.Write("Hello I'm " + "go-netty")
}

func (LoggerHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
    fmt.Println("go-netty:", "->", "handle read:", message)
    // 交给下一个处理器处理(按照处理器的注册顺序, 此例下一个处理器应该是UpperHandler)
    ctx.HandleRead(message)
}

func (LoggerHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
    fmt.Println("go-netty:", "->", "inactive:", ctx.Channel().RemoteAddr(), ex)
    // 连接断开了，默认处理是关闭连接
    ctx.HandleInactive(ex)
}

type UpperHandler struct {}

func (UpperHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
    // 业务逻辑，将字符串大写化
    text := message.(string)
    upText := strings.ToUpper(text)
    // 写入返回结果给客户端
    ctx.Write(text + " -> " + upText)
}
```

使用<code>Netcat</code>发送消息  
```bash
$ echo -n -e "Hello Go-Netty\nhttps://go-netty.com\n" | nc 127.0.0.1 9527
Hello I'm go-netty
Hello Go-Netty -> HELLO GO-NETTY
https://go-netty.com -> HTTPS://GO-NETTY.COM
```
