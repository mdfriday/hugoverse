package caddy

import "testing"

func TestMapHugoverseDataPathToCaddySiteRoot(t *testing.T) {
	tests := []struct {
		dataDir, siteRoot, hugoversePath, want string
	}{
		{"/data", "/srv", "/data/enterprise", "/srv/enterprise"},
		{"/data", "/srv", "/data/publish/u1/mdf_sub_domain", "/srv/publish/u1/mdf_sub_domain"},
		{"/data", "/srv", "/data", "/srv"},
		{"/data", "/srv", "/var/www/x", "/var/www/x"},
		{"", "/srv", "/data/enterprise", "/data/enterprise"},
		{"/data", "", "/data/enterprise", "/data/enterprise"},
		{"/data", "/srv", "", ""},
	}
	for _, tt := range tests {
		got := MapHugoverseDataPathToCaddySiteRoot(tt.dataDir, tt.siteRoot, tt.hugoversePath)
		if got != tt.want {
			t.Errorf("MapHugoverseDataPathToCaddySiteRoot(%q,%q,%q) = %q; want %q",
				tt.dataDir, tt.siteRoot, tt.hugoversePath, got, tt.want)
		}
	}
}
