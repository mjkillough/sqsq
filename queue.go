package sqsq

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Queue struct {
	sqs *sqs.SQS
	url string
}

func NewQueue() *Queue {
	// TODO: Take this as input somehow. This is tied to local development!!
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String("localhost"),
			Credentials: credentials.NewStaticCredentials("1", "2", "3"),
		},
	}))

	return &Queue{
		sqs: sqs.New(sess, &aws.Config{
			Endpoint: aws.String("http://localhost:9324"),
			Region:   aws.String("localhost"),
		}),
		url: "http://localhost:9324/queue/default",
	}
}
