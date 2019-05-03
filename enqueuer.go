package sqsq

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type Enqueuer struct {
	q *Queue
}

func NewEnqueuer(queue *Queue) Enqueuer {
	return Enqueuer{
		q: queue,
	}
}

func (e *Enqueuer) Enqueue(worker string, arguments interface{}) error {
	json, err := json.Marshal(arguments)
	if err != nil {
		return errors.Wrapf(err, "sqsq: marshal JSON")
	}

	fmt.Println(json)

	msg := string(json)
	result, err := e.q.sqs.SendMessage(&sqs.SendMessageInput{
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"name": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: &worker,
			},
		},
		MessageBody: &msg,
		QueueUrl:    &e.q.url,
	})

	if err != nil {
		fmt.Println("Error", err)
		return errors.Wrapf(err, "sqsq: enqueuing")
	}

	fmt.Println("Success", *result.MessageId)

	return nil
}
