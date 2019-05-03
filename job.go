package sqsq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type Job struct {
	q   *Queue
	msg *sqs.Message
}

func (j *Job) ID() string {
	if j.msg.MessageId == nil {
		return "unknown"
	}
	return *j.msg.MessageId
}

func (j *Job) WorkerName() (string, bool) {
	attr := j.msg.MessageAttributes["name"]
	if attr == nil || *attr.DataType != "String" || attr.StringValue == nil {
		return "", false
	}
	return *attr.StringValue, true
}

// TODO: Also provide a `func Arguments() Arguments` that returns an object
// with functions like `ArgString(key) string`, similar to gocraft/work?
func (j *Job) UnmarshalArgs(out interface{}) error {
	err := json.Unmarshal([]byte(*j.msg.Body), out)
	if err != nil {
		return errors.Wrapf(err, "sqsq: unmarshalling args")
	}
	return nil
}

func (j *Job) deleteMessage() error {
	_, err := j.q.sqs.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &j.q.url,
		ReceiptHandle: j.msg.ReceiptHandle,
	})
	if err != nil {
		return errors.Wrapf(err, "sqsq: deleting message")
	}
	return nil
}

func (j *Job) retry(err error) {
	// TODO: Exponential back-off of retries.
	// Possible implementation:
	//  - Add "retry" message attribute storing retry count.
	//  - Add "error" message attribute capturing last error for debug reasons
	//  - Whenever retry() is called, updateVisibilityTimeout(exponentialBackoff(retrycount))
	fmt.Printf("retry %#v: %v", j, err)
}

func (j *Job) updateVisibilityTimeout(worker *Worker, interval time.Duration) {
	// TODO
	fmt.Printf("updateVisibilityTimeout %#v", j)
}
