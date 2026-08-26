package sarp

import "testing"

func TestDecodeSnapshotSupportsSARPEnvelopeAndStringCounters(t *testing.T) {
	body := []byte(`{"message":"OK","payload":{"id":1995,"name":"Warehouse","bank":"$1,234.00","employees":[{"character_id":"14381","character_name":"Ian","rank_id":903,"rank_name":"Owner","earnings":"2,500","duty_time":12,"historical_duty_time":30}]}}`)
	snapshot, err := decodeSnapshot(body)
	if err != nil {
		t.Fatalf("decodeSnapshot() error = %v", err)
	}
	if snapshot.BusinessID != 1995 || snapshot.Bank != 1234 || len(snapshot.Employees) != 1 ||
		snapshot.Employees[0].CharacterID != 14381 || snapshot.Employees[0].RankID != 903 ||
		snapshot.Employees[0].Earnings != 2500 || snapshot.Employees[0].Name != "Ian" {
		t.Fatalf("decodeSnapshot() = %#v", snapshot)
	}
}

func TestDecodeRanksSupportsEnvelopeAndStringCounters(t *testing.T) {
	ranks, err := decodeRanks([]byte(`{"payload":[{"id":"903","name":"Chief Executive Officer","permissions":"255","paycheck":"$500"}]}`))
	if err != nil {
		t.Fatalf("decodeRanks() error = %v", err)
	}
	if len(ranks) != 1 || ranks[0].ID != 903 || ranks[0].Permissions != 255 || ranks[0].Paycheck != 500 {
		t.Fatalf("decodeRanks() = %#v", ranks)
	}
}

func TestDecodeSnapshotRejectsMissingBusiness(t *testing.T) {
	if _, err := decodeSnapshot([]byte(`{"payload":{"name":"unknown"}}`)); err == nil {
		t.Fatal("decodeSnapshot() error = nil")
	}
}
