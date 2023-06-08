package domains

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func getValues(t *testing.T, domain *LuaDomain) (values []string) {
	t.Helper()
	for {
		value, ok, err := domain.Next()
		require.NoError(t, err)
		if !ok {
			break
		}
		values = append(values, value)
	}
	return values
}

func TestIntLuaDomain(t *testing.T) {
	l := lua.NewState()
	t.Cleanup(l.Close)
	RegisterModule(l)

	err := l.DoString(`domains = require("domains")`)
	require.NoError(t, err)
	tests := []*struct {
		name      string
		code      string
		want      []string
		wantErr   bool
		skipReset bool
	}{
		{
			name:    "int range",
			code:    `id = domains.Int:new(0, 1, 10)`,
			want:    []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			wantErr: false,
		},
		{
			name:    "int negative",
			code:    `id = domains.Int:new(0, 1, 1, true)`,
			want:    []string{"0", "1", "-1"},
			wantErr: false,
		},
		{
			name: "time",
			code: `id = domains.Time:new{
				now = os.time{
					year = 2023,
					month = 1,
					day = 3,
					isdst = true,
					hour = 0,
				},
				step = 1 + 60*60*24,
				top = 5,
				allow_negative = true,
			}`,
			want: []string{
				"2023-01-03T00:00:00Z",
				"2023-01-04T00:00:01Z",
				// отнимается один день И одна секунда
				"2023-01-01T23:59:59Z",
				"2023-01-05T00:00:02Z",
				"2022-12-31T23:59:58Z",
			},
			wantErr: false,
		},
		{
			name:    "float range",
			code:    `id = domains.Float:new(1, 1, 3)`,
			want:    []string{"1", "2", "3"},
			wantErr: false,
		},
		{
			name: "float negative",
			code: `id = domains.Float:new(0, 0.5, 3.0, true)`,
			want: []string{
				"0",
				"0.5", "-0.5", "1", "-1",
				"1.5", "-1.5", "2", "-2",
				"2.5", "-2.5", "3", "-3",
			},
			wantErr: false,
		},
		{
			name:    "bool",
			code:    `id = domains.Bool()`,
			want:    []string{"True", "False"},
			wantErr: false,
		},
		{
			name: "bool again",
			code: `
			id = domains.Bool()
			id:next()
			id2 = domains.Bool()
			id2:next()
			id2:next()
			id2:next()
			`,
			skipReset: true,
			want:      []string{"False"},
			wantErr:   false,
		},
		{
			name: "uuid",
			code: `
			id = domains.Bool()
			id:next()
			id:next()
			local uuid = domains.UUID:new()
			local value = uuid:next()
			if not value then error("no uuid value") end
			`,
			skipReset: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			err := l.DoString(tt.code)
			r.NoError(err)

			domain, err := NewLuaDomain(l, l.GetGlobal("id"))
			if err != nil {
				r.True(tt.wantErr, "err: %+v", err)
			}

			values := getValues(t, domain)
			r.Equal(tt.want, values)

			if tt.skipReset {
				return
			}
			err = domain.Reset()
			r.NoError(err)

			values = getValues(t, domain)
			r.Equal(tt.want, values, "after reset")
		})
	}
}
