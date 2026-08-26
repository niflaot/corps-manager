package sarp

import "testing"

func TestDecodeSnapshotSupportsSARPEnvelopeAndStringCounters(t *testing.T) {
	body := []byte(`{"message":"OK","payload":{"id":1995,"name":"Warehouse","bank":"$1,234.00","employees":[{"character_id":"14381","character_name":"Ian","rank_name":"Owner","earnings":"2,500","duty_time":12,"historical_duty_time":30}]}}`)
	snapshot, err := decodeSnapshot(body)
	if err != nil {
		t.Fatalf("decodeSnapshot() error = %v", err)
	}
	if snapshot.BusinessID != 1995 || snapshot.Bank != 1234 || len(snapshot.Employees) != 1 ||
		snapshot.Employees[0].CharacterID != 14381 || snapshot.Employees[0].Earnings != 2500 || snapshot.Employees[0].Name != "Ian" {
		t.Fatalf("decodeSnapshot() = %#v", snapshot)
	}
}

func TestDecodeSnapshotRejectsMissingBusiness(t *testing.T) {
	if _, err := decodeSnapshot([]byte(`{"payload":{"name":"unknown"}}`)); err == nil {
		t.Fatal("decodeSnapshot() error = nil")
	}
}
