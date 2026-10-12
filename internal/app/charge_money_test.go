package app

import (
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
)

func TestAllocFavourableF0(t *testing.T) {
	// 例 1：全退、无优惠。G=20，退全额。
	fref, frem, pay, rnew, ok := allocFavourable(domain.FavourableAllocProportional, 2000, 0, 2000)
	if !ok || fref != 0 || frem != 0 || pay != 2000 || rnew != 0 {
		t.Fatalf("full %+v", []int64{fref, frem, pay, rnew})
	}
	// 例 2：部分退、无优惠。G=40，退 10。
	_, _, pay, rnew, ok = allocFavourable(domain.FavourableAllocProportional, 4000, 0, 1000)
	if !ok || pay != 1000 || rnew != 3000 {
		t.Fatalf("partial pay=%d rnew=%d", pay, rnew)
	}
	// 三种策略在 F=0 时一致。
	for _, st := range []string{domain.FavourableAllocDiscountOnRemain, domain.FavourableAllocDiscountFirst} {
		_, _, pay, rnew, ok = allocFavourable(st, 4000, 0, 1000)
		if !ok || pay != 1000 || rnew != 3000 {
			t.Fatalf("%s pay=%d rnew=%d", st, pay, rnew)
		}
	}
}

func TestAllocFavourableProportional(t *testing.T) {
	// 例 4：G=40，F=4，退毛额 10 → Fref=1，退款 9，新单 27。
	fref, frem, pay, rnew, ok := allocFavourable(domain.FavourableAllocProportional, 4000, 400, 1000)
	if !ok || fref != 100 || frem != 300 || pay != 900 || rnew != 2700 || pay+rnew != 3600 {
		t.Fatalf("proportional %d %d %d %d ok=%v", fref, frem, pay, rnew, ok)
	}
	// 例 3：全退有优惠。G=20，F=5，退款 15。
	fref, frem, pay, rnew, ok = allocFavourable(domain.FavourableAllocProportional, 2000, 500, 2000)
	if !ok || fref != 500 || frem != 0 || pay != 1500 || rnew != 0 {
		t.Fatalf("full favour %d %d %d %d", fref, frem, pay, rnew)
	}
	// 例 5：G=50，F=5，退毛额 10 → 退款 9，新单 36。
	_, frem, pay, rnew, ok = allocFavourable(domain.FavourableAllocProportional, 5000, 500, 1000)
	if !ok || frem != 400 || pay != 900 || rnew != 3600 {
		t.Fatalf("multi %d %d %d", frem, pay, rnew)
	}
}

func TestPortionCentsConserves(t *testing.T) {
	ref, ok := portionCents(4000, 1000, 4000)
	if !ok || ref != 1000 {
		t.Fatalf("ref %d", ref)
	}
	rem, ok := portionCents(4000, 3000, 4000)
	if !ok || rem != 3000 || ref+rem != 4000 {
		t.Fatalf("rem %d", rem)
	}
}
