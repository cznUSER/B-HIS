package app

import "testing"

func TestLineFeeCents(t *testing.T) {
	fee, ok := lineFeeCents(2000, 1000)
	if !ok || fee != 2000 {
		t.Fatalf("DRUG-A x2 = %d %v", fee, ok)
	}
	fee, ok = lineFeeCents(1000, 800)
	if !ok || fee != 800 {
		t.Fatalf("DRUG-B x1 = %d %v", fee, ok)
	}
	if _, ok := lineFeeCents(0, 1000); ok {
		t.Fatal("zero amount must fail")
	}
	if _, ok := amountMilli(0); ok {
		t.Fatal("amount 0")
	}
	if _, ok := amountMilli(-1); ok {
		t.Fatal("amount -1")
	}
	cents, ok := priceCents(10)
	if !ok || cents != 1000 {
		t.Fatalf("price %d %v", cents, ok)
	}
}
