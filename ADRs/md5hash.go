package main

import (
	"fmt"
	"os"

	"github.com/mdfriday/hugoverse/pkg/hash"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run md5hash.go <input_string>")
		fmt.Println("\nExample:")
		fmt.Println("  go run md5hash.go 'hello@example.com'")
		fmt.Println("  go run md5hash.go 'my-subdomain'")
		os.Exit(1)
	}

	input := os.Args[1]
	
	// 计算完整的 MD5 hash
	fullHash := hash.MD5(input)
	
	// 获取前 16 个字符
	shortHash := fullHash[:16]
	
	fmt.Printf("Input:      %s\n", input)
	fmt.Printf("Full MD5:   %s\n", fullHash)
	fmt.Printf("Short MD5:  %s (first 16 chars)\n", shortHash)
}

