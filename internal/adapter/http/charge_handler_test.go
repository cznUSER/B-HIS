package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestChargeHTTP(t *testing.T) {
	ts, st, issuer := newOrderHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	doc := domain.User{
		ID: "00000000-0000-4000-8000-0000000000b2", Username: "doctor2", DisplayName: "门诊医生",
		Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
	}
	st.Add(doc)
	docTok := bearerFor(t, issuer, doc)
	patientID := mustCreatePatient(t, ts.URL, tok)
	registerID := mustRegisterID(t, ts.URL, tok, patientID)
	orderID := mustCommitOrder(t, ts.URL, tok, registerID, drugA(2))

	t.Run("unauthorized", func(t *testing.T) {
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+orderID+`"}`, nil)
		if code != 401 || decodeErr(t, body).Error.Code != "UNAUTHORIZED" {
			t.Fatalf("create %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/x/pay", `{"payType":"CASH","amount":20}`, nil)
		if code != 401 {
			t.Fatalf("pay %d %s", code, body)
		}
	})

	t.Run("doctor cannot create", func(t *testing.T) {
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+orderID+`"}`, authHdr(docTok))
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("%d %s", code, body)
		}
	})

	t.Run("preview save is refused and commit preview is unsaved", func(t *testing.T) {
		saveID := mustSaveOrder(t, ts.URL, tok, registerID, drugA(1))
		code, body := getAuth(t, ts.URL+"/api/v1/orders/"+saveID+"/charge-preview", tok)
		if code != 409 || decodeErr(t, body).Error.Code != "ILLEGAL_STATE_TRANSITION" {
			t.Fatalf("save %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/orders/"+orderID+"/charge-preview", tok)
		if code != 200 {
			t.Fatalf("preview %d %s", code, body)
		}
		pv := decodeCharge(t, body)
		if pv.ChargeID != "" || pv.SquareNo != "" || pv.PaidAt != nil || pv.Status != "NORMAL" {
			t.Fatalf("unsaved %+v", pv)
		}
		if pv.RegisterID != registerID || pv.EncounterID != registerID || pv.PatientID != patientID {
			t.Fatalf("bind %+v", pv)
		}
		if !near(pv.TotalFee, 20) || !near(pv.SquareFee, 20) || !near(pv.FavourableFee, 0) || len(pv.Items) != 1 || !near(pv.Items[0].Qty, 2) {
			t.Fatalf("%+v", pv)
		}
	})

	var chargeID, itemID string
	t.Run("create and pay cash", func(t *testing.T) {
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+orderID+`"}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("create %d %s", code, body)
		}
		ch := decodeCharge(t, body)
		if ch.ChargeID == "" || ch.SquareNo == "" || ch.PaidAt != nil || ch.Status != "NORMAL" {
			t.Fatalf("%+v", ch)
		}
		if !near(ch.TotalFee, 20) || !near(ch.SquareFee, 20) || !near(ch.FavourableFee, 0) || len(ch.Items) != 1 {
			t.Fatalf("%+v", ch)
		}
		if ch.Items[0].ChargeItemID == "" || !ch.Items[0].CanBack || !near(ch.Items[0].Qty, 2) {
			t.Fatalf("%+v", ch.Items[0])
		}
		chargeID = ch.ChargeID
		itemID = ch.Items[0].ChargeItemID
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+orderID+`"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ILLEGAL_STATE_TRANSITION" {
			t.Fatalf("second %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/pay", `{"payType":"CASH","amount":15}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "CHARGE_AMOUNT_MISMATCH" {
			t.Fatalf("low %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/pay", `{"payType":"CASH","amount":25}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "CHARGE_AMOUNT_MISMATCH" {
			t.Fatalf("high %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/pay", `{"payType":"CASH","amount":-1}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "INVALID_AMOUNT" {
			t.Fatalf("neg %d %s", code, body)
		}
		still := decodeCharge(t, mustGet(t, ts.URL+"/api/v1/charges/"+chargeID, tok))
		if still.PaidAt != nil {
			t.Fatalf("paid early %+v", still)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/pay", `{"payType":"CASH","amount":20}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("pay %d %s", code, body)
		}
		pay := decodePayment(t, body)
		if pay.Direction != "CHARGE" || pay.PayType != "CASH" || pay.Status != "SUCCEEDED" || !near(pay.Amount, 20) || pay.ChargeID != chargeID {
			t.Fatalf("%+v", pay)
		}
		if pay.PaidAt == nil || pay.SucceededAt == nil || *pay.PaidAt != *pay.SucceededAt {
			t.Fatalf("paidAt %+v", pay)
		}
		got := decodeCharge(t, mustGet(t, ts.URL+"/api/v1/charges/"+chargeID, tok))
		if got.Status != "NORMAL" || got.PaidAt == nil || !near(got.SquareFee, 20) || !near(got.FavourableFee, 0) {
			t.Fatalf("%+v", got)
		}
		ord := decodeOrder(t, mustGet(t, ts.URL+"/api/v1/orders/"+orderID, tok))
		if !ord.Charged || ord.Status != "COMMIT" {
			t.Fatalf("order %+v", ord)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/orders/"+orderID+"/abate", `{}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ORDER_CHARGED_NOT_ABATABLE" {
			t.Fatalf("abate %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/pay", `{"payType":"CASH","amount":20}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ILLEGAL_STATE_TRANSITION" {
			t.Fatalf("repay %d %s", code, body)
		}
		if n := countChargeAudit(st, "charge.create", "SUCCESS", chargeID); n != 1 {
			t.Fatalf("create audit %d", n)
		}
		if n := countChargeAudit(st, "charge.pay", "SUCCESS", chargeID); n != 1 {
			t.Fatalf("pay audit %d", n)
		}
	})

	t.Run("list filters", func(t *testing.T) {
		code, body := getAuth(t, ts.URL+"/api/v1/charges", tok)
		if code != 200 || !strings.Contains(string(body), `"items":[]`) {
			t.Fatalf("empty %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/charges?patientId="+patientID, tok)
		if code != 200 || !chargeListHas(decodeChargeList(t, body), chargeID) {
			t.Fatalf("patient %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/charges?encounterId="+registerID+"&visitDate=2026-10-12", tok)
		if code != 200 || !chargeListHas(decodeChargeList(t, body), chargeID) {
			t.Fatalf("encounter day %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/charges?registerId="+registerID+"&from=2026-10-12&to=2026-10-12", tok)
		if code != 200 || !chargeListHas(decodeChargeList(t, body), chargeID) {
			t.Fatalf("range %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/charges?patientId="+patientID+"&from=2026-10-13&to=2026-10-13", tok)
		if code != 200 || chargeListHas(decodeChargeList(t, body), chargeID) {
			t.Fatalf("other day %d %s", code, body)
		}
	})

	t.Run("full refund undispensed", func(t *testing.T) {
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+itemID+`","refundQty":2}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("refund %d %s", code, body)
		}
		if !strings.Contains(string(body), `"newSquare":null`) {
			t.Fatalf("newSquare %s", body)
		}
		res := decodeRefund(t, body)
		if res.AbatedCharge.Status != "ABATE" || res.AbatedCharge.SuccessorChargeID != nil {
			t.Fatalf("abated %+v", res.AbatedCharge)
		}
		if res.NewSquare != nil {
			t.Fatalf("square %+v", res.NewSquare)
		}
		if res.RefundPayment.Direction != "REFUND" || res.RefundPayment.Status != "SUCCEEDED" || !near(res.RefundPayment.Amount, 20) {
			t.Fatalf("pay %+v", res.RefundPayment)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+itemID+`","refundQty":2}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ALREADY_ABATED" {
			t.Fatalf("again %d %s", code, body)
		}
	})

	t.Run("partial refund F0", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(4))
		ch := mustChargePay(t, ts.URL, tok, oid, 40)
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","favourableAlloc":"PROPORTIONAL","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","refundQty":1}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("%d %s", code, body)
		}
		res := decodeRefund(t, body)
		if res.AbatedCharge.Status != "ABATE" || res.NewSquare == nil || !near(res.NewSquare.SquareFee, 30) || res.NewSquare.Status != "NORMAL" {
			t.Fatalf("%+v new=%+v", res.AbatedCharge, res.NewSquare)
		}
		if res.AbatedCharge.SuccessorChargeID == nil || *res.AbatedCharge.SuccessorChargeID != res.NewSquare.ChargeID {
			t.Fatalf("successor %+v", res.AbatedCharge)
		}
		if res.NewSquare.AbatedFromID == nil || *res.NewSquare.AbatedFromID != ch.ChargeID || !near(res.NewSquare.FavourableFee, 0) {
			t.Fatalf("from %+v", res.NewSquare)
		}
		if len(res.NewSquare.Items) != 1 || !near(res.NewSquare.Items[0].Qty, 3) {
			t.Fatalf("qty %+v", res.NewSquare.Items)
		}
		if !near(res.RefundPayment.Amount, 10) || res.RefundPayment.Direction != "REFUND" {
			t.Fatalf("refund pay %+v", res.RefundPayment)
		}
	})

	t.Run("multi line 18", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(1)+","+drugB(1))
		ch := mustChargePay(t, ts.URL, tok, oid, 18)
		if len(ch.Items) != 2 || !near(ch.TotalFee, 18) || !near(ch.SquareFee, 18) {
			t.Fatalf("%+v", ch)
		}
	})

	t.Run("encounter back and stock and pay recheck", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		code, body, _ := postJSON(t, ts.URL+"/api/v1/registers/"+reg+"/cancel", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("cancel %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/orders/"+oid+"/charge-preview", tok)
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("preview back %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+oid+`"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("create back %d %s", code, body)
		}

		reg = mustRegisterID(t, ts.URL, tok, patientID)
		oid = mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		if err := st.SetDrugOnHandMilli("DRUG-A", 1000); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+oid+`"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "INSUFFICIENT_STOCK" {
			t.Fatalf("stock %d %s", code, body)
		}
		if err := st.SetDrugOnHandMilli("DRUG-A", 100000); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+oid+`"}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("create %d %s", code, body)
		}
		ch := decodeCharge(t, body)
		if err := st.SetDrugOnHandMilli("DRUG-A", 0); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/pay", `{"payType":"CASH","amount":20}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "INSUFFICIENT_STOCK" {
			t.Fatalf("pay stock %d %s", code, body)
		}
		if decodeCharge(t, mustGet(t, ts.URL+"/api/v1/charges/"+ch.ChargeID, tok)).PaidAt != nil {
			t.Fatal("paid despite stock")
		}
		if err := st.SetDrugOnHandMilli("DRUG-A", 100000); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/registers/"+reg+"/cancel", `{}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("cancel2 %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/pay", `{"payType":"CASH","amount":20}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ENCOUNTER_REQUIRED" {
			t.Fatalf("pay back %d %s", code, body)
		}
	})

	t.Run("refund requires return then allows after return", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		ch := mustChargePay(t, ts.URL, tok, oid, 20)
		if err := st.SetOrderStatus(oid, domain.OrderStatusProc); err != nil {
			t.Fatal(err)
		}
		ref := `{"payType":"CASH","items":[{"chargeItemId":"` + ch.Items[0].ChargeItemID + `","refundQty":2}]}`
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", ref, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("proc %d %s", code, body)
		}
		if decodeCharge(t, mustGet(t, ts.URL+"/api/v1/charges/"+ch.ChargeID, tok)).Status != "NORMAL" {
			t.Fatal("status changed")
		}
		if err := st.SetDispenseMilli(ch.Items[0].ChargeItemID, 2000, 2000); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", ref, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), `"newSquare":null`) {
			t.Fatalf("after return %d %s", code, body)
		}
	})

	t.Run("explicit dispense blocks until returned", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		ch := mustChargePay(t, ts.URL, tok, oid, 20)
		if err := st.SetDispenseMilli(ch.Items[0].ChargeItemID, 2000, 0); err != nil {
			t.Fatal(err)
		}
		ref := `{"payType":"CASH","items":[{"chargeItemId":"` + ch.Items[0].ChargeItemID + `","refundQty":2}]}`
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", ref, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("%d %s", code, body)
		}
	})

	t.Run("can back and qty", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		ch := mustChargePay(t, ts.URL, tok, oid, 20)
		if err := st.SetChargeItemCanBack(ch.Items[0].ChargeItemID, false); err != nil {
			t.Fatal(err)
		}
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","refundQty":1}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ITEM_NOT_CAN_BACK" {
			t.Fatalf("can %d %s", code, body)
		}
		if err := st.SetChargeItemCanBack(ch.Items[0].ChargeItemID, true); err != nil {
			t.Fatal(err)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","refundQty":3}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_EXCEEDS_REFUNDABLE" {
			t.Fatalf("qty %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","refundQty":0}]}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "INVALID_AMOUNT" {
			t.Fatalf("zero %d %s", code, body)
		}
		if decodeCharge(t, mustGet(t, ts.URL+"/api/v1/charges/"+ch.ChargeID, tok)).Status != "NORMAL" {
			t.Fatal("status changed")
		}
	})

	t.Run("wechat stored without provider", func(t *testing.T) {
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(1))
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+oid+`"}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		ch := decodeCharge(t, body)
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/pay", `{"payType":"WECHAT","amount":10}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("%d %s", code, body)
		}
		pay := decodePayment(t, body)
		if pay.PayType != "WECHAT" || pay.Status != "SUCCEEDED" || !near(pay.Amount, 10) {
			t.Fatalf("%+v", pay)
		}
	})
}

func mustSaveOrder(t *testing.T, origin, tok, registerID, items string) string {
	t.Helper()
	code, body, _ := postJSON(t, origin+"/api/v1/orders", orderBody(registerID, items), authHdr(tok))
	if code != 201 {
		t.Fatalf("save %d %s", code, body)
	}
	return decodeOrder(t, body).OrderID
}

func mustCommitOrder(t *testing.T, origin, tok, registerID, items string) string {
	t.Helper()
	id := mustSaveOrder(t, origin, tok, registerID, items)
	code, body, _ := postJSON(t, origin+"/api/v1/orders/"+id+"/commit", `{}`, authHdr(tok))
	if code != 200 {
		t.Fatalf("commit %d %s", code, body)
	}
	dto := decodeOrder(t, body)
	if dto.Status != "COMMIT" {
		t.Fatalf("status %+v", dto)
	}
	return id
}

func mustChargePay(t *testing.T, origin, tok, orderID string, amount float64) chargeJSON {
	t.Helper()
	code, body, _ := postJSON(t, origin+"/api/v1/charges", `{"orderId":"`+orderID+`"}`, authHdr(tok))
	if code != 201 {
		t.Fatalf("charge %d %s", code, body)
	}
	ch := decodeCharge(t, body)
	code, body, _ = postJSON(t, origin+"/api/v1/charges/"+ch.ChargeID+"/pay", `{"payType":"CASH","amount":`+jsonNumber(amount)+`}`, authHdr(tok))
	if code != 200 {
		t.Fatalf("pay %d %s", code, body)
	}
	return ch
}

func mustGet(t *testing.T, url, tok string) []byte {
	t.Helper()
	code, body := getAuth(t, url, tok)
	if code != 200 {
		t.Fatalf("get %s %d %s", url, code, body)
	}
	return body
}

type chargeJSON struct {
	ChargeID          string           `json:"chargeId"`
	SquareNo          string           `json:"squareNo"`
	RegisterID        string           `json:"registerId"`
	EncounterID       string           `json:"encounterId"`
	PatientID         string           `json:"patientId"`
	TotalFee          float64          `json:"totalFee"`
	SquareFee         float64          `json:"squareFee"`
	FavourableFee     float64          `json:"favourableFee"`
	Status            string           `json:"status"`
	SuccessorChargeID *string          `json:"successorChargeId"`
	AbatedFromID      *string          `json:"abatedFromId"`
	PaidAt            *string          `json:"paidAt"`
	Items             []chargeItemJSON `json:"items"`
}

type chargeItemJSON struct {
	ChargeItemID string  `json:"chargeItemId"`
	Qty          float64 `json:"qty"`
	CanBack      bool    `json:"canBack"`
}

type paymentJSON struct {
	PaymentID   string  `json:"paymentId"`
	ChargeID    string  `json:"chargeId"`
	Direction   string  `json:"direction"`
	PayType     string  `json:"payType"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	PaidAt      *string `json:"paidAt"`
	SucceededAt *string `json:"succeededAt"`
}

type refundJSON struct {
	AbatedCharge  chargeJSON  `json:"abatedCharge"`
	NewSquare     *chargeJSON `json:"newSquare"`
	RefundPayment paymentJSON `json:"refundPayment"`
}

func decodeCharge(t *testing.T, body []byte) chargeJSON {
	t.Helper()
	var dto chargeJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return dto
}

func decodePayment(t *testing.T, body []byte) paymentJSON {
	t.Helper()
	var dto paymentJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return dto
}

func decodeRefund(t *testing.T, body []byte) refundJSON {
	t.Helper()
	var dto refundJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return dto
}

func decodeChargeList(t *testing.T, body []byte) []chargeJSON {
	t.Helper()
	var list struct {
		Items []chargeJSON `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return list.Items
}

func countChargeAudit(st *memory.Store, action, outcome, resourceID string) int {
	n := 0
	for _, e := range st.Audits() {
		if e.Action == action && e.Outcome == outcome && e.ResourceType == "Charge" && e.ResourceID == resourceID {
			n++
		}
	}
	return n
}

func chargeListHas(items []chargeJSON, id string) bool {
	for _, it := range items {
		if it.ChargeID == id {
			return true
		}
	}
	return false
}
