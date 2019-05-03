package main

import "sqsq"

func main() {
	enqueuer := sqsq.NewEnqueuer(sqsq.NewQueue())

	err := enqueuer.Enqueue("ping", &map[string]int{"one": 2})
	if err != nil {
		panic(err)
	}
}
