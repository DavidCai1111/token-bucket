package bucket_test

import (
	"fmt"
	"time"

	bucket "github.com/DavidCai1993/token-bucket"
)

func Example() {
	tb := bucket.New(time.Second, 1000)

	tb.Take(10)
	ok := tb.TakeMaxDuration(1000, 20*time.Second)

	fmt.Println(ok)
	// -> true

	tb.WaitMaxDuration(1000, 10*time.Second)

	fmt.Println(ok)
	// -> false

	tb.Wait(1000)
}
