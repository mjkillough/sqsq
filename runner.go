package sqsq

import (
	"context"
	"fmt"
	"log"
	"sqsq/internal/heartbeat"
	"sqsq/internal/pool"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type MiddlewareFunc func(JobFunc) JobFunc

type JobFunc func(context.Context, *Job) error

type Worker struct {
	Fn                JobFunc
	VisibilityTimeout time.Duration
}

const (
	// Maximum number of messages that ReceiveMessages can fetch in one request.
	maxReceiveMessages = 10

	defaultConcurrency = 10

	defaultVisibilityInterval = 60 * time.Second
	defaultHeartbeatInterval  = 60 * time.Second
	defaultWaitInterval       = 20 * time.Second
)

type Reporter interface {
	Report(err error)
}

type Runner struct {
	q           *Queue
	registry    map[string]*Worker
	middlewares []MiddlewareFunc

	concurrency       int
	heartbeatInterval time.Duration
	waitInterval      time.Duration

	reporter Reporter

	stop    chan struct{}
	stopped chan struct{}
}

func NewRunner(queue *Queue, opts ...RunnerOption) Runner {
	runner := Runner{
		q:        queue,
		registry: make(map[string]*Worker),

		concurrency:       defaultConcurrency,
		heartbeatInterval: defaultHeartbeatInterval,
		waitInterval:      defaultWaitInterval,

		reporter: nil,

		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(&runner)
	}

	return runner
}

func (r *Runner) RegisterWorker(name string, worker *Worker) {
	if worker.VisibilityTimeout == 0 {
		worker.VisibilityTimeout = defaultVisibilityInterval
	}
	r.registry[name] = worker
}

func (r *Runner) Middleware(fn MiddlewareFunc) {
	r.middlewares = append(r.middlewares, fn)
}

func (r *Runner) Run() {
	p := pool.New(r.concurrency, maxReceiveMessages, r.dispatch)

	_, _ = <-r.stop
	p.Stop()
	close(r.stopped)
}

func (r *Runner) Stop() {
	close(r.stop)
	_, _ = <-r.stopped
}

func (r *Runner) dispatch(num int) []func() {
	jobs := r.fetch(num)

	fns := make([]func(), 0, len(jobs))
	for _, job := range jobs {
		fn := func() {
			workerName, ok := job.WorkerName()
			if !ok {
				job.retry(errors.New("sqsq: could not fetch name"))
				return
			}
			worker, ok := r.registry[workerName]
			if !ok {
				job.retry(errors.New("sqsq: unknown worker"))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), worker.VisibilityTimeout)
			defer cancel()

			// TODO: Think more carefully about how many heartbeats a job will see. Is there a race
			// between the last heartbeat and the visibility timeout expiring causing the visibility
			// to actually be VisibilityTimeout + HeartbeatInterval?
			beat := func() { job.updateVisibilityTimeout(worker, r.heartbeatInterval) }
			fn := r.wrapMiddleware(ctx, worker)

			heartbeat.Run(r.heartbeatInterval, beat, func() {
				err := fn(ctx, job)
				if err != nil {
					job.retry(err)
					return
				}

				err = job.deleteMessage()
				if err != nil {
					r.report(err)
				}
			})
		}

		fns = append(fns, fn)
	}

	return fns
}

func (r *Runner) wrapMiddleware(ctx context.Context, worker *Worker) JobFunc {
	// Apply middlewares in reverse order to registration:
	fn := worker.Fn
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		middleware := r.middlewares[i]
		fn = middleware(fn)
	}
	return fn
}

func (r *Runner) fetch(num int) []*Job {
	fmt.Printf("Trying to fetch %v...\n", num)

	// TODO: Investigate backoff in AWS client.
	resp, err := r.q.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
		MaxNumberOfMessages:   aws.Int64(int64(num)),
		AttributeNames:        []*string{aws.String("All")},
		MessageAttributeNames: []*string{aws.String("All")},
		QueueUrl:              &r.q.url,
		VisibilityTimeout:     aws.Int64(int64(r.heartbeatInterval.Seconds())),
		WaitTimeSeconds:       aws.Int64(int64(r.waitInterval.Seconds())),
	})
	if err != nil {
		r.report(errors.Wrapf(err, "sqsq: fetching"))
		return nil
	}
	if len(resp.Messages) == 0 {
		return nil
	}

	// If we received any jobs, return them immediately to start execution.
	jobs := make([]*Job, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		job := &Job{r.q, msg}
		fmt.Printf("Received job: %#v\n", job.msg)

		jobs = append(jobs, job)
	}

	return jobs
}

func (r *Runner) report(err error) {
	if r.reporter != nil {
		r.reporter.Report(err)
	} else {
		log.Printf("error: %v", err)
	}
}

type RunnerOption func(*Runner)

func WithConcurrency(concurrency int) RunnerOption {
	return func(r *Runner) {
		r.concurrency = concurrency
	}
}

func WithHeatbeatInterval(interval time.Duration) RunnerOption {
	return func(r *Runner) {
		r.heartbeatInterval = interval
	}
}

func WithWaitInterval(interval time.Duration) RunnerOption {
	return func(r *Runner) {
		r.waitInterval = interval
	}
}

func WithReporter(reporter Reporter) RunnerOption {
	return func(r *Runner) {
		r.reporter = reporter
	}
}
