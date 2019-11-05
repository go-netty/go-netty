# GO-NETTY

<!--[![GoDoc][1]][2]-->
[![license-Apache 2][3]][4]

[1]: https://godoc.org/github.com/go-netty/go-netty?status.svg
[2]: https://godoc.org/github.com/go-netty/go-netty
[3]: https://img.shields.io/badge/license-Apache%202-blue.svg
[4]: LICENSE

## Introduction (介绍)

go-netty is heavily inspired by [netty](https://github.com/netty/netty)  
go-netty 大量参考了netty的设计并融合Golang本身的协程特性而开发的一款高性能网络库

## Feature (特性)

* Extensible multi-transport protocol support, default support TCP, KCP, Websocket
* 可扩展多种传输协议，并且默认实现了 TCP, KCP, Websocket
* Extensible multi-codec support
* 可扩展多种解码器，默认实现了常见的编解码器

## TODO (待完成)

* test case
* docs
* examples

## Sample

```go

func main() {

    var bootstrap = netty.NewBootstrap()

    bootstrap.ChildInitializer(func(channel netty.Channel) {
        channel.Pipeline().
            AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)).
            AddLast(format.TextCodec()).
            AddLast(LogHandler{"Server"})
    })

    bootstrap.ClientInitializer(func(channel netty.Channel) {
        channel.Pipeline().
            AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)).
            AddLast(format.TextCodec()).
        AddLast(LogHandler{"Client"})
    })

    time.AfterFunc(time.Second, func() {
        _, err := bootstrap.Connect("tcp://127.0.0.1:6565", nil)
        utils.Assert(err)
    })

    bootstrap.
        Transport(tcp.New()).
        Listen("tcp://0.0.0.0:6565").
        RunForever(os.Kill, os.Interrupt)
}

type LogHandler struct {
    role string
}

func (l LogHandler) HandleActive(ctx netty.ActiveContext) {
    fmt.Println(l.role, "->", "active:", ctx.Channel().RemoteAddr())
    ctx.Write("Hello Im " + l.role)
    ctx.HandleActive()
}

func (l LogHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
    fmt.Println(l.role, "->", "handle read:", message)
    ctx.HandleRead(message)
}

func (l LogHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
    fmt.Println(l.role, "->", "inactive:", ctx.Channel().RemoteAddr(), ex)
    ctx.HandleInactive(ex)
}

/*
Output:

Server -> active: 127.0.0.1:22142
Client -> active: 127.0.0.1:6565
Client -> handle read: Hello I'm Server
Server -> handle read: Hello I'm Client

*/
```
