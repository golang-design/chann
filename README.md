# chann [![PkgGoDev](https://pkg.go.dev/badge/golang.design/x/chann)](https://pkg.go.dev/golang.design/x/chann) ![](https://changkun.de/urlstat?mode=github&repo=golang-design/chann)

a unified representation of buffered, unbuffered, and unbounded channels in Go

```
import "golang.design/x/chann"
```

This package requires Go 1.18.

## Usage


Unbuffered channel:

```
ch := chann.New[func()](chann.Cap(0))            // unbufferd, capacity 0
```

Unbuffered channel:

```
ch := chann.New[map[int]float64](chann.Cap(100)) // buffered, capacity 100
```

Unbounded channel

```
ch := chann.New[int]() // unbounded, capacity unlimited
```

Send and receive:

```
ch.In() <- 42
println(<-ch.Out()) // 42
```

Access properties:

```
ch.ApproxLen() // the (approx. of) length of the channel
ch.Cap()       // the capacity of the channel
```

See https://golang.design/research/ultimate-channel for more details of the motivation of this abstraction.

## License


MIT | &copy; 2021 The [golang.design](https://golang.design) Initiative Authors, written by [Changkun Ou](https://changkun.de).