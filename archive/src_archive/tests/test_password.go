package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "admin123"
	hash := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"
	
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", hash)
	
	// 测试密码验证
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Printf("Password verification failed: %v\n", err)
	} else {
		fmt.Println("Password verification successful!")
	}
	
	// 生成新的哈希用于比较
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Failed to generate hash: %v\n", err)
		return
	}
	
	fmt.Printf("New hash for same password: %s\n", string(newHash))
	
	// 验证新生成的哈希
	err = bcrypt.CompareHashAndPassword(newHash, []byte(password))
	if err != nil {
		fmt.Printf("New hash verification failed: %v\n", err)
	} else {
		fmt.Println("New hash verification successful!")
	}
}
