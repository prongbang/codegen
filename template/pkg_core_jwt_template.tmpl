package core

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

const (
	UID = "uid"
)

const (
	ExpiresDurationAccessToken  = time.Hour * 12
	ExpiresDurationRefreshToken = time.Hour * 24
)

type TokenData struct {
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func GenerateTokenInfo(tokenData *TokenData, accessTokenExpiresDuration time.Duration, refreshTokenExpiresDuration time.Duration, secretKey string) (*TokenInfo, error) {
	accessToken, err := GenerateJwtToken(tokenData, accessTokenExpiresDuration, secretKey)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateJwtToken(tokenData, refreshTokenExpiresDuration, secretKey)
	if err != nil {
		return nil, err
	}
	return &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func GenerateJwtToken(tokenData *TokenData, expiresDuration time.Duration, secretKey string) (string, error) {
	currentDate := time.Now()
	tokenData.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   tokenData.UserID,
		ExpiresAt: jwt.NewNumericDate(currentDate.Add(expiresDuration)),
		IssuedAt:  jwt.NewNumericDate(currentDate),
		NotBefore: jwt.NewNumericDate(currentDate),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	return token.SignedString([]byte(secretKey))
}

func ValidateJwtToken(tokenString string, secretKey string) (*TokenData, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenData{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parse error: %w", err)
	}

	claims, ok := token.Claims.(*TokenData)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func GetTokenData(tokenString string) (*TokenData, error) {
	claims := &TokenData{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func ExtractToken(bearToken string) string {
	if strings.Contains(bearToken, "Bearer") {
		strArr := strings.Split(bearToken, " ")
		if len(strArr) == 2 {
			return strings.TrimSpace(strArr[1])
		}
	}
	return strings.TrimSpace(bearToken)
}
