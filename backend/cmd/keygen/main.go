package main

import (
	"flag"
	"fmt"
	"os"

	"google-ai-proxy/internal/auth"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 并初始化 JWT 密钥，确保签发密钥与运行中的后端一致（否则兑换验签失败）
	godotenv.Load(".env", "../.env", "../../backend/.env")
	auth.InitSecretKey()

	credits := flag.Int("credits", 200, "Credits to add to user account")
	flag.Parse()

	if *credits <= 0 {
		fmt.Println("Credits must be positive")
		os.Exit(1)
	}

	key, err := auth.GenerateLicenseKey(*credits)
	if err != nil {
		fmt.Printf("Error generating key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated License Key:")
	fmt.Println(key)
}
