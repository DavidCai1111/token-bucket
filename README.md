# token-bucket
[![Build Status](https://travis-ci.org/DavidCai1993/ratelimit.svg?branch=master)](https://travis-ci.org/DavidCai1993/ratelimit)
[![Coverage Status](https://coveralls.io/repos/github/DavidCai1993/ratelimit/badge.svg?branch=master)](https://coveralls.io/github/DavidCai1993/ratelimit?branch=master)

A [token bucket](https://en.wikipedia.org/wiki/Token_bucket) implementation based on multi goroutines which is safe to use under concurrency environments.

## Installation

```
go get -u github.com/DavidCai1993/token-bucket
```

## Documentation

API documentation can be found here: https://godoc.org/github.com/DavidCai1993/token-bucket

## Usage

```go
import (
  bucket "github.com/DavidCai1993/token-bucket"
)
```

```go
tb := bucket.New(time.Second, 1000)

tb.Take(10)
ok := tb.TakeMaxDuration(1000, 20*time.Second)

fmt.Println(ok)
// -> true

tb.WaitMaxDuration(1000, 10*time.Second)

fmt.Println(ok)
// -> false

tb.Wait(1000)
```
