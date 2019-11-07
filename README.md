# GO-NETTY

[![GoDoc][1]][2] [![license-Apache 2][3]][4] [![Go Report Card][5]][6] [![Build Status][9]][10] [![Coverage Statusd][11]][12]

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

## Introduction (介绍)

go-netty is heavily inspired by [netty](https://github.com/netty/netty)  
go-netty 大量参考了netty的设计并融合Golang本身的协程特性而开发的一款高性能网络库

## Feature (特性)

* Extensible multi-transport protocol support, default support TCP, KCP, Websocket
* 可扩展多种传输协议，并且默认实现了 TCP, KCP, Websocket
* Extensible multi-codec support
* 可扩展多种解码器，默认实现了常见的编解码器
* Based on responsibility chain model
* 基于责任链模型的流程控制

## TODO (待完成)

* test case
* docs
* examples

## Examples (例子)

* [chat_server (基于websocket的聊天室)](./example/chat_server/main.go)  
* [file_server (基于http的文件浏览器)](./example/file_server/main.go)  
* [tcp_server (自义定tcp服务器)](./example/tcp_server/main.go)  

## Usage (使用)

> 创建bootstrap, 用于提供服务或者对外建立连接

```go
var bootstrap = netty.NewBootstrap()
```

> 配置服务连接的处理器 (同样还有一个ClientInitializer 对应客户端连接处理器配置)

```go
bootstrap.ChildInitializer(func(channel netty.Channel) {
    channel.Pipeline().
        // 按照自定义协议解码帧（2字节的长度字段）
        AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)).
        // 消息内容为文本格式(可自定义为 json，protobuf 等编解码器)
        AddLast(format.TextCodec()).
        // 处理消息
        AddLast(LogHandler{"Server"})
})
```

> 配置服务器(客户端)所使用的传输协议

```go
bootstrap.Transport(tcp.New())
```

> 开始监听端口并开始提供服务，直到收到指定信号后退出

```go
bootstrap.Listen("tcp://0.0.0.0:6565").RunForever(os.Kill, os.Interrupt)
```

> LogHandler 处理器

```go
type LogHandler struct {
    role string
}

func (l LogHandler) HandleActive(ctx netty.ActiveContext) {
    fmt.Println(l.role, "->", "active:", ctx.Channel().RemoteAddr())

    // 给对端发送一条消息，将进入如下流程（视编解码配置）
    // Text -> TextCodec -> LengthFieldCodec   -> Channel.Write
    // 文本     文本编码      组装协议格式（长度字段）     网络发送
    ctx.Write("Hello I'm " + l.role)

    // 向后续的handler传递控制权
    // 如果是最后一个handler或者需要中断请求可以不用调用
    ctx.HandleActive()
}

func (l LogHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
    fmt.Println(l.role, "->", "handle read:", message)

    // 向后续的handler传递控制权
    // 如果是最后一个handler或者需要中断请求可以不用调用
    ctx.HandleRead(message)
}

func (l LogHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
    fmt.Println(l.role, "->", "inactive:", ctx.Channel().RemoteAddr(), ex)

    // 向后续的handler传递控制权
    // 如果是最后一个handler或者需要中断请求可以不用调用
    ctx.HandleInactive(ex)
}
```
