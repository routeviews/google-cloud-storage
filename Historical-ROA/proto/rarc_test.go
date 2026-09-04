package proto

import (
	"testing"
)

func TestResultsFromDB(t *testing.T) {
	tests := []struct {
		name                string
		item                *ResultsFromDB
		wantASN             string
		wantPrefix          string
		wantMaxlen          int32
		wantTa              string
		wantMask            int32
		wantFullprefix      string
		wantFullprefixrange string
		wantUnixLen         int
		wantRFC3339Len      int
	}{
		{
			name: "Populated",
			item: &ResultsFromDB{
				ASN:             "AS15169",
				Prefix:          "8.8.8.0",
				Maxlen:          24,
				Ta:              "arin",
				Mask:            24,
				Fullprefix:      "8.8.8.0/24",
				Fullprefixrange: "8.8.8.0/24",
				Unixtimearr:     []int64{1234567890},
				RFC3339Timearr:  []string{"2026-01-01T00:00:00Z"},
			},
			wantASN:             "AS15169",
			wantPrefix:          "8.8.8.0",
			wantMaxlen:          24,
			wantTa:              "arin",
			wantMask:            24,
			wantFullprefix:      "8.8.8.0/24",
			wantFullprefixrange: "8.8.8.0/24",
			wantUnixLen:         1,
			wantRFC3339Len:      1,
		},
		{
			name:                "Nil",
			item:                nil,
			wantASN:             "",
			wantPrefix:          "",
			wantMaxlen:          0,
			wantTa:              "",
			wantMask:            0,
			wantFullprefix:      "",
			wantFullprefixrange: "",
			wantUnixLen:         0,
			wantRFC3339Len:      0,
		},
	}

	for _, tc := range tests {
		if got := tc.item.GetASN(); got != tc.wantASN {
			t.Errorf("[%s] GetASN() = %v, want %v", tc.name, got, tc.wantASN)
		}
		if got := tc.item.GetPrefix(); got != tc.wantPrefix {
			t.Errorf("[%s] GetPrefix() = %v, want %v", tc.name, got, tc.wantPrefix)
		}
		if got := tc.item.GetMaxlen(); got != tc.wantMaxlen {
			t.Errorf("[%s] GetMaxlen() = %v, want %v", tc.name, got, tc.wantMaxlen)
		}
		if got := tc.item.GetTa(); got != tc.wantTa {
			t.Errorf("[%s] GetTa() = %v, want %v", tc.name, got, tc.wantTa)
		}
		if got := tc.item.GetMask(); got != tc.wantMask {
			t.Errorf("[%s] GetMask() = %v, want %v", tc.name, got, tc.wantMask)
		}
		if got := tc.item.GetFullprefix(); got != tc.wantFullprefix {
			t.Errorf("[%s] GetFullprefix() = %v, want %v", tc.name, got, tc.wantFullprefix)
		}
		if got := tc.item.GetFullprefixrange(); got != tc.wantFullprefixrange {
			t.Errorf("[%s] GetFullprefixrange() = %v, want %v", tc.name, got, tc.wantFullprefixrange)
		}
		if got := len(tc.item.GetUnixtimearr()); got != tc.wantUnixLen {
			t.Errorf("[%s] len(GetUnixtimearr()) = %v, want %v", tc.name, got, tc.wantUnixLen)
		}
		if got := len(tc.item.GetRFC3339Timearr()); got != tc.wantRFC3339Len {
			t.Errorf("[%s] len(GetRFC3339Timearr()) = %v, want %v", tc.name, got, tc.wantRFC3339Len)
		}
		if tc.item != nil {
			tc.item.ProtoMessage()
			_ = tc.item.String()
			_ = tc.item.ProtoReflect()
			_, _ = tc.item.Descriptor()
			tc.item.Reset()
		}
	}
}

func TestResultsFromDBRFC3339(t *testing.T) {
	tests := []struct {
		name                string
		item                *ResultsFromDBRFC3339
		wantASN             string
		wantPrefix          string
		wantMaxlen          int32
		wantTa              string
		wantMask            int32
		wantTimeLen         int
		wantFullprefix      string
		wantFullprefixrange string
	}{
		{
			name: "Populated",
			item: &ResultsFromDBRFC3339{
				ASN:             "AS15169",
				Prefix:          "8.8.8.0",
				Maxlen:          24,
				Ta:              "arin",
				Mask:            24,
				Time:            []string{"2026-01-01T00:00:00Z"},
				Fullprefix:      "8.8.8.0/24",
				Fullprefixrange: "8.8.8.0/24",
			},
			wantASN:             "AS15169",
			wantPrefix:          "8.8.8.0",
			wantMaxlen:          24,
			wantTa:              "arin",
			wantMask:            24,
			wantTimeLen:         1,
			wantFullprefix:      "8.8.8.0/24",
			wantFullprefixrange: "8.8.8.0/24",
		},
		{
			name:                "Nil",
			item:                nil,
			wantASN:             "",
			wantPrefix:          "",
			wantMaxlen:          0,
			wantTa:              "",
			wantMask:            0,
			wantTimeLen:         0,
			wantFullprefix:      "",
			wantFullprefixrange: "",
		},
	}

	for _, tc := range tests {
		if got := tc.item.GetASN(); got != tc.wantASN {
			t.Errorf("[%s] GetASN() = %v, want %v", tc.name, got, tc.wantASN)
		}
		if got := tc.item.GetPrefix(); got != tc.wantPrefix {
			t.Errorf("[%s] GetPrefix() = %v, want %v", tc.name, got, tc.wantPrefix)
		}
		if got := tc.item.GetMaxlen(); got != tc.wantMaxlen {
			t.Errorf("[%s] GetMaxlen() = %v, want %v", tc.name, got, tc.wantMaxlen)
		}
		if got := tc.item.GetTa(); got != tc.wantTa {
			t.Errorf("[%s] GetTa() = %v, want %v", tc.name, got, tc.wantTa)
		}
		if got := tc.item.GetMask(); got != tc.wantMask {
			t.Errorf("[%s] GetMask() = %v, want %v", tc.name, got, tc.wantMask)
		}
		if got := len(tc.item.GetTime()); got != tc.wantTimeLen {
			t.Errorf("[%s] len(GetTime()) = %v, want %v", tc.name, got, tc.wantTimeLen)
		}
		if got := tc.item.GetFullprefix(); got != tc.wantFullprefix {
			t.Errorf("[%s] GetFullprefix() = %v, want %v", tc.name, got, tc.wantFullprefix)
		}
		if got := tc.item.GetFullprefixrange(); got != tc.wantFullprefixrange {
			t.Errorf("[%s] GetFullprefixrange() = %v, want %v", tc.name, got, tc.wantFullprefixrange)
		}
		if tc.item != nil {
			tc.item.ProtoMessage()
			_ = tc.item.String()
			_ = tc.item.ProtoReflect()
			_, _ = tc.item.Descriptor()
			tc.item.Reset()
		}
	}
}

func TestResultArr(t *testing.T) {
	tests := []struct {
		name       string
		item       *ResultArr
		wantResLen int
	}{
		{
			name: "Populated",
			item: &ResultArr{
				Results: []*ResultsFromDB{
					{ASN: "AS15169"},
				},
			},
			wantResLen: 1,
		},
		{
			name:       "Nil",
			item:       nil,
			wantResLen: 0,
		},
	}

	for _, tc := range tests {
		if got := len(tc.item.GetResults()); got != tc.wantResLen {
			t.Errorf("[%s] len(GetResults()) = %v, want %v", tc.name, got, tc.wantResLen)
		}
		if tc.item != nil {
			tc.item.ProtoMessage()
			_ = tc.item.String()
			_ = tc.item.ProtoReflect()
			_, _ = tc.item.Descriptor()
			tc.item.Reset()
		}
	}
}
