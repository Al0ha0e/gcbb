package net

import "testing"

func TestSign(t *testing.T) {
	sb := &NaiveAuthHandler{}
	sb.Sign()
}
