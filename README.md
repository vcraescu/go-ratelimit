# Rate limiter ![](https://github.com/vcraescu/go-ratelimit/actions/workflows/go.yml/badge.svg)

### Example

```go
package main

import (
	"fmt"
	"github.com/vcraescu/go-ratelimit"
	"log"
	"time"
)

func worker(i int) {
	fmt.Printf("worker: %d\n", i)
}

func main() {
	limiter := ratelimit.NewLimiter(10, ratelimit.WithPer(time.Second))

	for i := 0; i < 50; i++ {
		i := i

		limiter.Run(func() {
			worker(i)
		})
	}

	if err := limiter.Wait(); err != nil {
		log.Fatalln(err)
	}
}
```