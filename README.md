# chann [![PkgGoDev](https://pkg.go.dev/badge/golang.design/x/chann)](https://pkg.go.dev/golang.design/x/chann) ![](https://changkun.de/urlstat?mode=github&repo=golang-design/chann)

a unified representation of buffered, unbuffered, and unbounded channels in Go

```
import "golang.design/x/chann"
```

This package requires Go 1.18.

## Usage

Different types of channels:

```
ch := chann.New[int]()                           // unbounded, capacity unlimited
ch := chann.New[func()](chann.Cap(0))            // unbufferd, capacity 0
ch := chann.New[map[int]float64](chann.Cap(100)) // buffered,  capacity 100
```

Send and receive operations:

```
ch.In() <- 42
<-ch.Out() // 42
```

Channel properties:

```
ch.ApproxLen() // an (approx. of) length of the channel
ch.Cap()       // the capacity of the channel
```

See https://golang.design/research/ultimate-channel for more details of
the motivation of this abstraction.

## License


MIT | &copy; 2021 The [golang.design](https://golang.design) Initiative Authors, written by [Changkun Ou](https://changkun.de).