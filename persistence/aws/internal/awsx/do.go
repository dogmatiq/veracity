package awsx

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/request"
)

// Do executes an AWS API request.
//
// dec is a decorator function that mutates the request before it is sent.
func Do[In, Out any](
	ctx context.Context,
	fn func(context.Context, *In, ...request.Option) (Out, error),
	dec func(*In) []request.Option,
	in *In,
	options ...request.Option,
) (out Out, err error) {
	if dec != nil {
		options = append(options, dec(in)...)
	}

	return fn(ctx, in, options...)
}
