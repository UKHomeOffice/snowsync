package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/in"
)

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return in.Handle(req)
}

func main() {
	lambda.Start(handler)
}
