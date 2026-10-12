package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestDispenseHTTP(t *testing.T) {
	ts, st, issuer := newOrderHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	doc := domain.User{
		ID: "00000000-0000-4000-8000-0000000000d2", Username: "doctor-disp", DisplayName: "门诊医生",
		Roles: []domain.Role{domain.RoleDoctor}, Status: domain.StatusActive,
	}
	st.Add(doc)
	docTok := bearerFor(t, issuer, doc)
	patientID := mustCreatePatient(t, ts.URL, tok)
	resetStock(t, st)

	t.Run("inventory seed and auth", func(t *testing.T) {
		code, body := getAuth(t, ts.URL+"/api/v1/inventory/items", "")
		if code != 401 {
			t.Fatalf("auth %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/inventory/items", docTok)
		if code != 403 || decodeErr(t, body).Error.Code != "FORBIDDEN" {
			t.Fatalf("doctor %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/inventory/items", tok)
		if code != 200 {
			t.Fatalf("list %d %s", code, body)
		}
		items := decodeInventory(t, body)
		if len(items) != 2 {
			t.Fatalf("seed %+v", items)
		}
		if items[0].SkuCode != "DRUG-A" || items[0].ItemNo != "DRUG-A" || !near(items[0].QtyOnHand, 100) || items[0].Status != "ACTIVE" {
			t.Fatalf("a %+v", items[0])
		}
		if items[1].SkuCode != "DRUG-B" || !near(items[1].QtyOnHand, 100) {
			t.Fatalf("b %+v", items[1])
		}
		code, body = getAuth(t, ts.URL+"/api/v1/inventory/items?skuCode=DRUG-A", tok)
		got := decodeInventory(t, body)
		if code != 200 || len(got) != 1 || got[0].ItemNo != "DRUG-A" {
			t.Fatalf("sku %d %+v", code, got)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/inventory/items?itemNo=TREAT-1", tok)
		if code != 200 || len(decodeInventory(t, body)) != 0 {
			t.Fatalf("treat %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{}`, nil)
		if code != 401 {
			t.Fatalf("dispense auth %d %s", code, body)
		}
	})

	var orderID, chargeID, itemID, dispenseID, dispenseItemID string
	t.Run("pay does not deduct then dispense does", func(t *testing.T) {
		resetStock(t, st)
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		orderID = mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","items":[{"orderItemId":"x","dispenseQty":2}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "NOT_CHARGED" {
			t.Fatalf("unpaid %d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 100) {
			t.Fatal("stock moved")
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","items":[]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "EMPTY_DISPENSE_ITEMS" {
			t.Fatalf("empty %d %s", code, body)
		}
		ch := mustChargePay(t, ts.URL, tok, orderID, 20)
		chargeID = ch.ChargeID
		itemID = ch.Items[0].ChargeItemID
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 100) {
			t.Fatal("pay deducted")
		}
		ord := decodeOrder(t, mustGet(t, ts.URL+"/api/v1/orders/"+orderID, tok))
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","items":[{"chargeItemId":"`+itemID+`","dispenseQty":0}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DISPENSE_EXCEEDS_AVAILABLE" {
			t.Fatalf("zero %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","chargeId":"`+chargeID+`","items":[{"orderItemId":"`+ord.Items[0].OrderItemID+`","dispenseQty":3}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "DISPENSE_EXCEEDS_AVAILABLE" {
			t.Fatalf("over %d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 100) || decodeOrder(t, mustGet(t, ts.URL+"/api/v1/orders/"+orderID, tok)).Status != "COMMIT" {
			t.Fatal("mutated")
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/orders/"+orderID+"/dispense", `{"items":[{"chargeItemId":"`+itemID+`","dispenseQty":2}]}`, authHdr(docTok))
		if code != 403 {
			t.Fatalf("doctor %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","chargeId":"`+chargeID+`","items":[{"chargeItemId":"`+itemID+`","orderItemId":"`+ord.Items[0].OrderItemID+`","dispenseQty":2}]}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("dispense %d %s", code, body)
		}
		dto := decodeDispense(t, body)
		if dto.DispenseID == "" || dto.DispenseNo == "" || dto.Status != "DONE" || dto.OrderStatus != "PROC" || len(dto.Items) != 1 {
			t.Fatalf("%+v", dto)
		}
		line := dto.Items[0]
		if !near(line.DispensedQty, 2) || !near(line.ChargedQty, 2) || !near(line.AvailableQty, 0) || line.LineDispenseStatus != "FULL" || !near(line.StockQty, 98) {
			t.Fatalf("%+v", line)
		}
		dispenseID = dto.DispenseID
		dispenseItemID = line.DispenseItemID
		if decodeOrder(t, mustGet(t, ts.URL+"/api/v1/orders/"+orderID, tok)).Status != "PROC" {
			t.Fatal("order")
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 98) {
			t.Fatal("stock")
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+orderID+`","items":[{"chargeItemId":"`+itemID+`","dispenseQty":1}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ALREADY_DISPENSED" {
			t.Fatalf("again %d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 98) {
			t.Fatal("double deduct")
		}
		got := decodeDispense(t, mustGet(t, ts.URL+"/api/v1/dispenses/"+dispenseID, tok))
		if got.DispenseID != dispenseID || !near(got.Items[0].DispensedQty, 2) {
			t.Fatalf("get %+v", got)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/dispenses?patientId="+patientID+"&orderId="+orderID+"&chargeId="+chargeID+"&registerId="+reg+"&visitDate=2026-10-12", tok)
		if code != 200 || !dispenseListHas(decodeDispenseList(t, body), dispenseID) {
			t.Fatalf("list %d %s", code, body)
		}
		code, body = getAuth(t, ts.URL+"/api/v1/dispenses", tok)
		if code != 200 || len(decodeDispenseList(t, body)) != 0 {
			t.Fatalf("empty filter %d %s", code, body)
		}
	})

	t.Run("insufficient stock leaves rows unchanged", func(t *testing.T) {
		resetStock(t, st)
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		ch := mustChargePay(t, ts.URL, tok, oid, 20)
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 100) {
			t.Fatal("pay deducted")
		}
		if err := st.SetDrugOnHandMilli("DRUG-A", 1000); err != nil {
			t.Fatal(err)
		}
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":2}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "INSUFFICIENT_STOCK" {
			t.Fatalf("%d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 1) {
			t.Fatalf("stock %v", stockQty(t, ts.URL, tok, "DRUG-A"))
		}
		if decodeOrder(t, mustGet(t, ts.URL+"/api/v1/orders/"+oid, tok)).Status != "COMMIT" {
			t.Fatal("became proc")
		}
		code, body = getAuth(t, ts.URL+"/api/v1/dispenses?orderId="+oid, tok)
		if code != 200 || len(decodeDispenseList(t, body)) != 0 {
			t.Fatalf("slip %d %s", code, body)
		}
	})

	t.Run("partial then full", func(t *testing.T) {
		resetStock(t, st)
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(4))
		ch := mustChargePay(t, ts.URL, tok, oid, 40)
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":1.5}]}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("partial %d %s", code, body)
		}
		first := decodeDispense(t, body)
		if first.Items[0].LineDispenseStatus != "PARTIAL" || !near(first.Items[0].AvailableQty, 2.5) || !near(first.Items[0].DispensedQty, 1.5) {
			t.Fatalf("%+v", first.Items[0])
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 98.5) {
			t.Fatalf("stock %v", stockQty(t, ts.URL, tok, "DRUG-A"))
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/orders/"+oid+"/dispense", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":2.5}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("alias %d %s", code, body)
		}
		second := decodeDispense(t, body)
		if second.OrderStatus != "PROC" || second.Items[0].LineDispenseStatus != "FULL" || !near(second.Items[0].AvailableQty, 0) {
			t.Fatalf("%+v", second)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 96) {
			t.Fatalf("stock %v", stockQty(t, ts.URL, tok, "DRUG-A"))
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":1}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "ALREADY_DISPENSED" {
			t.Fatalf("full %d %s", code, body)
		}
	})

	t.Run("return restocks and unblocks refund", func(t *testing.T) {
		ref := `{"payType":"CASH","items":[{"chargeItemId":"` + itemID + `","refundQty":2}]}`
		code, body, _ := postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/refund", ref, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("refund %d %s", code, body)
		}
		before := stockQty(t, ts.URL, tok, "DRUG-A")
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses/"+dispenseID+"/return", `{"reason":"  ","items":[{"dispenseItemId":"`+dispenseItemID+`","returnQty":2}]}`, authHdr(tok))
		if code != 400 || decodeErr(t, body).Error.Code != "RETURN_REASON_REQUIRED" {
			t.Fatalf("reason %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses/"+dispenseID+"/return", `{"reason":"患者要求","items":[{"dispenseItemId":"`+dispenseItemID+`","returnQty":3}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "RETURN_EXCEEDS_DISPENSE" {
			t.Fatalf("over %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses/"+dispenseID+"/return", `{"reason":"患者要求","items":[{"dispenseItemId":"missing","returnQty":1}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "NOT_DISPENSED" {
			t.Fatalf("missing %d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), before) {
			t.Fatal("stock moved")
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses/"+dispenseID+"/return", `{"reason":"患者要求","items":[{"dispenseItemId":"`+dispenseItemID+`","returnQty":1}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("part %d %s", code, body)
		}
		part := decodeDispense(t, body)
		if part.Status != "PARTIAL_RETURNED" || part.ReturnReason == nil || *part.ReturnReason != "患者要求" || !near(part.Items[0].ReturnedQty, 1) {
			t.Fatalf("%+v", part)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/refund", ref, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("still %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses/"+dispenseID+"/return", `{"reason":"患者要求","items":[{"dispenseItemId":"`+dispenseItemID+`","returnQty":1}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("rest %d %s", code, body)
		}
		done := decodeDispense(t, body)
		if done.Status != "RETURNED" || done.Items[0].LineDispenseStatus != "NONE" || !near(done.Items[0].AvailableQty, 2) {
			t.Fatalf("%+v", done)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), before+2) {
			t.Fatalf("restock %v before %v", stockQty(t, ts.URL, tok, "DRUG-A"), before)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+chargeID+"/refund", ref, authHdr(tok))
		if code != 200 || !strings.Contains(string(body), `"newSquare":null`) {
			t.Fatalf("refund after %d %s", code, body)
		}
	})

	t.Run("never dispensed line can refund while a drug line cannot", func(t *testing.T) {
		resetStock(t, st)
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2)+`,{"itemNo":"TREAT-1","itemType":"TREAT","itemName":"演示诊查费","amount":1,"price":15}`)
		ch := mustChargePay(t, ts.URL, tok, oid, 35)
		var drugID, treatID string
		for _, it := range ch.Items {
			if near(it.Qty, 2) {
				drugID = it.ChargeItemID
			}
			if near(it.Qty, 1) {
				treatID = it.ChargeItemID
			}
		}
		if drugID == "" || treatID == "" {
			t.Fatalf("%+v", ch.Items)
		}
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+drugID+`","dispenseQty":2}]}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("dispense %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+drugID+`","refundQty":2}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("drug %d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+ch.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+treatID+`","refundQty":1}]}`, authHdr(tok))
		if code != 200 {
			t.Fatalf("treat %d %s", code, body)
		}
		res := decodeRefund(t, body)
		if res.NewSquare == nil || len(res.NewSquare.Items) != 1 {
			t.Fatalf("square %+v", res.NewSquare)
		}
		nextID := res.NewSquare.Items[0].ChargeItemID
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges/"+res.NewSquare.ChargeID+"/refund", `{"payType":"CASH","items":[{"chargeItemId":"`+nextID+`","refundQty":2}]}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "REFUND_REQUIRES_RETURN" {
			t.Fatalf("successor %d %s", code, body)
		}
	})

	t.Run("charge sees the same on hand", func(t *testing.T) {
		resetStock(t, st)
		if err := st.SetDrugOnHandMilli("DRUG-A", 2000); err != nil {
			t.Fatal(err)
		}
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(2))
		ch := mustChargePay(t, ts.URL, tok, oid, 20)
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 2) {
			t.Fatal("pre-deduct")
		}
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":2}]}`, authHdr(tok))
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 0) {
			t.Fatal(stockQty(t, ts.URL, tok, "DRUG-A"))
		}
		reg = mustRegisterID(t, ts.URL, tok, patientID)
		oid = mustCommitOrder(t, ts.URL, tok, reg, drugA(1))
		code, body, _ = postJSON(t, ts.URL+"/api/v1/charges", `{"orderId":"`+oid+`"}`, authHdr(tok))
		if code != 409 || decodeErr(t, body).Error.Code != "INSUFFICIENT_STOCK" {
			t.Fatalf("charge %d %s", code, body)
		}
	})

	t.Run("idempotency does not deduct twice", func(t *testing.T) {
		resetStock(t, st)
		reg := mustRegisterID(t, ts.URL, tok, patientID)
		oid := mustCommitOrder(t, ts.URL, tok, reg, drugA(1))
		ch := mustChargePay(t, ts.URL, tok, oid, 10)
		hdr := authHdr(tok)
		hdr["Idempotency-Key"] = "dispense-key-1"
		payload := `{"orderId":"` + oid + `","items":[{"chargeItemId":"` + ch.Items[0].ChargeItemID + `","dispenseQty":1}]}`
		code, body, _ := postJSON(t, ts.URL+"/api/v1/dispenses", payload, hdr)
		first := decodeDispense(t, body)
		if code != 201 {
			t.Fatalf("%d %s", code, body)
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", payload, hdr)
		second := decodeDispense(t, body)
		if code != 201 || second.DispenseID != first.DispenseID {
			t.Fatalf("replay %d %s", code, body)
		}
		if !near(stockQty(t, ts.URL, tok, "DRUG-A"), 99) {
			t.Fatalf("stock %v", stockQty(t, ts.URL, tok, "DRUG-A"))
		}
		code, body, _ = postJSON(t, ts.URL+"/api/v1/dispenses", `{"orderId":"`+oid+`","items":[{"chargeItemId":"`+ch.Items[0].ChargeItemID+`","dispenseQty":0.5}]}`, hdr)
		if code != 409 || decodeErr(t, body).Error.Code != "IDEMPOTENCY_KEY_CONFLICT" {
			t.Fatalf("conflict %d %s", code, body)
		}
	})
}

func resetStock(t *testing.T, st *memory.Store) {
	t.Helper()
	if err := st.SetDrugOnHandMilli("DRUG-A", 100000); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDrugOnHandMilli("DRUG-B", 100000); err != nil {
		t.Fatal(err)
	}
}

func stockQty(t *testing.T, origin, tok, sku string) float64 {
	t.Helper()
	code, body := getAuth(t, origin+"/api/v1/inventory/items?skuCode="+sku, tok)
	if code != 200 {
		t.Fatalf("stock %d %s", code, body)
	}
	items := decodeInventory(t, body)
	if len(items) != 1 {
		t.Fatalf("%+v", items)
	}
	return items[0].QtyOnHand
}

type dispenseJSON struct {
	DispenseID   string             `json:"dispenseId"`
	DispenseNo   string             `json:"dispenseNo"`
	OrderID      string             `json:"orderId"`
	Status       string             `json:"status"`
	OrderStatus  string             `json:"orderStatus"`
	ReturnReason *string            `json:"returnReason"`
	Items        []dispenseItemJSON `json:"items"`
}

type dispenseItemJSON struct {
	DispenseItemID     string  `json:"dispenseItemId"`
	DispensedQty       float64 `json:"dispensedQty"`
	ChargedQty         float64 `json:"chargedQty"`
	ReturnedQty        float64 `json:"returnedQty"`
	AvailableQty       float64 `json:"availableQty"`
	StockQty           float64 `json:"stockQty"`
	LineDispenseStatus string  `json:"lineDispenseStatus"`
}

type inventoryJSON struct {
	SkuCode   string  `json:"skuCode"`
	ItemNo    string  `json:"itemNo"`
	QtyOnHand float64 `json:"qtyOnHand"`
	Status    string  `json:"status"`
}

func decodeDispense(t *testing.T, body []byte) dispenseJSON {
	t.Helper()
	var dto dispenseJSON
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return dto
}

func decodeDispenseList(t *testing.T, body []byte) []dispenseJSON {
	t.Helper()
	var list struct {
		Items []dispenseJSON `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return list.Items
}

func decodeInventory(t *testing.T, body []byte) []inventoryJSON {
	t.Helper()
	var list struct {
		Items []inventoryJSON `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("%v %s", err, body)
	}
	return list.Items
}

func dispenseListHas(items []dispenseJSON, id string) bool {
	for _, it := range items {
		if it.DispenseID == id {
			return true
		}
	}
	return false
}
