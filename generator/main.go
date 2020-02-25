package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	generator "github.com/storozhukBM/allocator/generator/internal"
)

func main() {
	typeNames := flag.String("type", "", "comma-separated list of type names; must be set")
	var dirName string
	flag.StringVar(&dirName, "dir", ".", "working directory; must be set")

	flag.Parse()
	if len(*typeNames) == 0 {
		log.Fatalf("the flag -type must be set")
	}
	if len(dirName) == 0 {
		log.Fatalf("the flag -dir must be set")
	}
	types := strings.Split(*typeNames, ",")
	g := generator.NewGenerator()
	generationErr := g.RunGeneratorForTypes(dirName, types)
	if generationErr != nil {
		fmt.Printf("can't generate allocators: %v", generationErr)
		os.Exit(1)
	}
}
