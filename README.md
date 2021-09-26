# chann ![example workflow](https://github.com/golang-design/chann/actions/workflows/chann.yml/badge.svg) ![](https://changkun.de/urlstat?mode=github&repo=golang-design/chann)

a unified representation of buffered, unbuffered, and unbounded channels in Go

```go
import "golang.design/x/chann"
```

This package requires Go 1.18.

## Usage

Different types of channels:

```go
ch := chann.New[int]()                  // unbounded, capacity unlimited
ch := chann.New[func()](chann.Cap(0))   // unbufferd, capacity 0
ch := chann.New[string](chann.Cap(100)) // buffered,  capacity 100
```

Send and receive operations:

```go
ch.In() <- 42
println(<-ch.Out()) // 42
```

Close operation:

```go
ch.Close()
```

Channel properties:

```go
ch.ApproxLen() // an (approx. of) length of the channel
ch.Cap()       // the capacity of the channel
```

See https://golang.design/research/ultimate-channel for more details of
the motivation of this abstraction.

## License


MIT | &copy; 2021 The [golang.design](https://golang.design) Initiative Authors, written by [Changkun Ou](https://changkun.de).