package main

import (
	"os"
	"strconv"
	"strings"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitByComma(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func getenvBool(key string) bool {
	val, _ := strconv.ParseBool(os.Getenv(key))
	return val
}
