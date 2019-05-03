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
