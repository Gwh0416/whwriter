package service

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
)

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) GenerateCode() string {
	code := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += fmt.Sprintf("%d", n)
	}
	return code
}

func (s *EmailService) SendVerificationCode(email, code string) error {
	log.Printf("[EMAIL] sending verification code %s to %s", code, email)
	return nil
}
