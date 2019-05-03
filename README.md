# SQSQ — SQS (Job) Q(ueue)

[![CircleCI](https://circleci.com/gh/mjkillough/sqsq.svg?style=svg&circle-token=39d7b55dba74c18b3130d2c2a553aabc917f1695)](https://circleci.com/gh/mjkillough/sqsq)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg)](http://godoc.deliveroo.net/github.com/mjkillough/sqsq)

A simple job queue for Go, built on top of SQS.

## Design

 - Workers are the code that runs Jobs:
    - Jobs are SQS messages.
    - Jobs can take a JSON document as arguments. The Worker can interpret this however it wants.
    - Have a configurable `VisibilityTimeout` (timeout after which the job will be killed/retried).
 - The `Runner`:
    - Has a (configurable) amount of concurrency.
    - Uses `ReceiveMessages` to long-poll SQS for messages, possibly receiving a batch of messages to improve throughput.
    - Has a fixed amount of concurrency, fetching at most as many jobs as there is space in the worker pool.
    - Uses a small initial `VisibilityTimeout` on fetched messages, so that jobs are quickly retried if a process dies.
    - Wraps each worker in a heartbeat process, which will periodically update the `VisibilityTimeout` of the message in small increments, up to the total `VisibilityTimeout` of the worker.
    - Provides a simple middleware functionality.
 - Ideally the implementation will be simpler (and a little less magic) than gocraft/work, making it easy to maintain.

## Usage

Run `alpine-sqs` locally:

```
docker run --name alpine-sqs -p 9324:9324 -p 9325:9325 -d roribio16/alpine-sqs:latest
```

`./cmd/server/`:

```go
package main

import (
    "context"
    "fmt"
    "sqsq"
)

func main() {
    runner := sqsq.NewRunner(sqsq.NewQueue())

    runner.Middleware(func(inner sqsq.JobFunc) sqsq.JobFunc {
        return func(ctx context.Context, job *sqsq.Job) error {
            fmt.Printf("Middleware job: %#v\n", job)
            return inner(ctx, job)
        }
    })

    runner.RegisterWorker("ping", &sqsq.Worker{
        Fn: func(ctx context.Context, job *sqsq.Job) error {
            // Could also provide a simpler (but more magic) args interface,
            // like gocraft/work.
            var args struct {
                One int `json:"one"`
            }
            err := job.UnmarshalArgs(&args)
            if err != nil {
                return err
            }

            fmt.Printf("Hello from ping: %v\n", args.One)
            return nil
        },
    })

    runner.Run()
}
```

`./cmd/client/`:

```go
package main

import "sqsq"

func main() {
    enqueuer := sqsq.NewEnqueuer(sqsq.NewQueue())

    // We could also pass a struct{} in as arguments: anything that can be
    // marshaled to JSON.
    err := enqueuer.Enqueue("ping", &map[string]int{"one": 2})
    if err != nil {
        panic(err)
    }
}
```

## TODO

There's lots to do before this could even be considered a finished proof-of-concept:

 - Complete TODOs in code, in particular the exponential retry.
 - Statsd middleware, similar to Sidekiq's for job count/error rate.
 - tracer-go middleware
 - Optional Statsd extension that reports queue length/retries etc.
 - Enqueuer.EnqueueIn() using visibility timeout/delay?

## Possibly not TODO

When I set out, I tried to keep this simple and was planning on not implementing:

 - FIFO queues/uniqueness.
 - Multiple queues per Runner. Create multiple runners.
 - cron: Use external cron package and persistent process.

