// Copyright (c) OpenMMLab. All rights reserved.

package main

import (
	"fmt"
	"os"

	"deeptrace/pkg/client"
)

func main() {
	deeptracex := client.NewDeepTracexCommand()

	if err := deeptracex.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
