package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/authjwt"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
)

const defaultUserID = "11111111-1111-1111-1111-111111111111"

func main() {
	userID := flag.String("user-id", defaultUserID, "JWT user_id/sub claim")
	role := flag.String("role", "operator", "JWT role claim")
	ttlOverride := flag.String("ttl", "", "JWT TTL, for example 1h or 15m")
	secretOverride := flag.String("secret", "", "JWT secret override")
	jti := flag.String("jti", "", "JWT ID override")
	flag.Parse()

	cfg, err := config.Load("../../environment/.env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	secret := strings.TrimSpace(*secretOverride)
	if secret == "" {
		secret = cfg.Security.JWTSecret
	}

	ttlRaw := strings.TrimSpace(*ttlOverride)
	if ttlRaw == "" {
		ttlRaw = cfg.Security.JWTTTL
	}
	ttl, err := time.ParseDuration(ttlRaw)
	if err != nil {
		log.Fatalf("invalid JWT TTL %q: %v", ttlRaw, err)
	}

	token, _, err := authjwt.Generate(secret, *userID, *role, *jti, ttl, time.Now())
	if err != nil {
		log.Fatalf("failed to generate JWT: %v", err)
	}
	fmt.Println(token)
}
