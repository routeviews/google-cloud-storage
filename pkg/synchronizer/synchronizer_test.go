package synchronizer

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func parseTime(t *testing.T, ts string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestSpannedMonths(t *testing.T) {
	tests := []struct {
		desc       string
		start      time.Time
		end        time.Time
		wantMonths []string
	}{
		{
			desc:       "start and end times are in the same month",
			start:      parseTime(t, "2006-01-02T15:04:05+00:00"),
			end:        parseTime(t, "2006-01-03T15:04:05+00:00"),
			wantMonths: []string{"2006.01"},
		},
		{
			desc:       "spans several months",
			start:      parseTime(t, "2006-10-02T15:04:05+00:00"),
			end:        parseTime(t, "2007-03-03T15:04:05+00:00"),
			wantMonths: []string{"2006.10", "2006.11", "2006.12", "2007.01", "2007.02", "2007.03"},
		},
		{
			desc:       "spans from last second of a month to the first second of another",
			start:      parseTime(t, "2006-10-31T23:59:59+00:00"),
			end:        parseTime(t, "2006-12-01T00:00:00+00:00"),
			wantMonths: []string{"2006.10", "2006.11", "2006.12"},
		},
		{
			desc:  "start is after end",
			start: parseTime(t, "2006-01-02T15:04:05+00:00"),
			end:   parseTime(t, "2006-01-01T15:04:05+00:00"),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := spannedMonths(test.start, test.end)
			if diff := cmp.Diff(test.wantMonths, got, cmpopts.SortSlices(func(a, b string) bool {
				return a < b
			})); diff != "" {
				t.Errorf("spanned months mismatched: (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestTimeFromFileName(t *testing.T) {
	tests := []struct {
		desc     string
		filename string
		wantTime time.Time
		wantErr  bool
	}{
		{
			desc:     "valid path",
			filename: "route-views.linx/bgpdata/2022.04/UPDATES/updates.20220427.1900.bz2",
			wantTime: parseTime(t, "2022-04-27T19:00:00+00:00"),
		},
		{
			desc:     "valid path started with a slash",
			filename: "/route-views.linx/bgpdata/2022.04/UPDATES/updates.20220427.1900.bz2",
			wantTime: parseTime(t, "2022-04-27T19:00:00+00:00"),
		},
		{
			desc:     "invalid filename 1",
			filename: "/route-views.linx/bgpdata/2022.04/UPDATES/updates.20220427.1900",
			wantErr:  true,
		},
		{
			desc:     "invalid filename 2",
			filename: "updates.20220427.1900",
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := timeFromFilename(test.filename)
			if gotErr := err != nil; gotErr != test.wantErr {
				t.Errorf("timeFromFilename(%s): err %v; wantErr %v", test.filename, err, test.wantErr)
			}
			if got.Sub(test.wantTime) != 0 {
				t.Errorf("timeFromFilename(%s): %v; want %v", test.filename, got, test.wantTime)
			}
		})
	}
}

func TestPrepareDir(t *testing.T) {
	// prepareDir(dir map[string][]string) int
	tests := []struct {
		desc      string
		dir       map[string][]string
		wantDir   map[string][]string
		wantTotal int
	}{
		{
			desc: "only one collector was found",
			dir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
				},
			},
			wantDir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
				},
			},
			wantTotal: 2,
		},
		{
			desc: "directory without valid files",
			dir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
				},
				"dnszone": nil,
				"bgpmon":  {},
			},
			wantDir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
				},
			},
			wantTotal: 2,
		},
		{
			desc: "multiple collectors",
			dir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
				},
				"route-views4": {
					"route-views4/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
					"route-views4/bgpdata/2022.05/UPDATES/updates.20220111.0930.bz2",
					"route-views4/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
				},
			},
			wantDir: map[string][]string{
				"route-views6": {
					"route-views6/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
					"route-views6/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
				},
				"route-views4": {
					"route-views4/bgpdata/2022.05/UPDATES/updates.20220111.0930.bz2",
					"route-views4/bgpdata/2022.05/UPDATES/updates.20220511.0930.bz2",
					"route-views4/bgpdata/2022.05/UPDATES/updates.20221011.0930.bz2",
				},
			},
			wantTotal: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotDir, gotTotal := prepareDir(test.dir)
			if diff := cmp.Diff(test.wantDir, gotDir); diff != "" {
				t.Errorf("prepareDir mismatched: (-want, +got):\n%s", diff)
			} else if test.wantTotal != gotTotal {
				t.Errorf("prepareDir: %d files, want %d", gotTotal, test.wantTotal)
			}
		})
	}
}
