// Copyright (c) OpenMMLab. All rights reserved.

package textparser

import (
	"context"
)

type Interface[T any] interface {
	Parse(ctx context.Context, inputs []string) (T, error)
}

// Generic processing function that returns results of a specific type
func ParseWithType[T any](ctx context.Context, parser Interface[T], inputs []string) (T, error) {
	return parser.Parse(ctx, inputs)
}
