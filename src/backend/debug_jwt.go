package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// 使用从登录获得的JWT令牌
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzM5Mjk2NDQsImlzcyI6IndlYi12MiIsInN1YiI6IjEiLCJ1c2VyX2lkIjoxfQ.kOJi2CNLyMLcQ5c3Rh99JHB6HUkOk2fTGGCzI3_8mfM"
	
	// 如果命令行提供了token，使用它
	if len(os.Args) > 1 {
		token = os.Args[1]
	}
	
	// 分解JWT令牌
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Fatal("Invalid JWT token format")
	}

	// 解码header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		log.Fatal("Failed to decode header:", err)
	}

	// 解码payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Fatal("Failed to decode payload:", err)
	}

	fmt.Println("=== JWT Header ===")
	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		log.Fatal("Failed to parse header JSON:", err)
	}
	headerJSON, _ := json.MarshalIndent(header, "", "  ")
	fmt.Println(string(headerJSON))

	fmt.Println("\n=== JWT Payload ===")
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Fatal("Failed to parse payload JSON:", err)
	}
	payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(payloadJSON))

	fmt.Println("\n=== Analysis ===")
	if userID, exists := payload["user_id"]; exists {
		fmt.Printf("user_id field found: %v (type: %T)\n", userID, userID)
	} else {
		fmt.Println("user_id field NOT found in payload")
	}

	if sub, exists := payload["sub"]; exists {
		fmt.Printf("sub field found: %v (type: %T)\n", sub, sub)
	} else {
		fmt.Println("sub field NOT found in payload")
	}
}
